package dbcrypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/glasskube/pkg/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	. "github.com/onsi/gomega"
)

func testKey(fill byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, crypto.MinKeyLen))
}

// useKeyring points the process wide keyring at a test keyring, so the [String] and [Bytes] scan
// paths can run without an initialized environment.
func useKeyring(t *testing.T) {
	t.Helper()
	keyring, err := crypto.ParseKeyring(testKey(1))
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	previous := keys
	keys = keyring
	t.Cleanup(func() { keys = previous })
}

// scanned is what the read expression of a column yields for a stored value: the length and the
// bytes of the data it is bound to, followed by the value itself.
func scanned(t *testing.T, c Column, scope []uuid.UUID, stored []byte) []byte {
	t.Helper()
	aad, err := c.aad(scope)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	return append(append([]byte{byte(len(aad))}, aad...), stored...)
}

var (
	secretValue   = NewColumn("Secret", "value", "customer_organization_id", "organization_id")
	smtpUsername  = NewColumn("CustomEmailConfiguration", "smtp_username", "organization_id")
	smtpPassword  = NewColumn("CustomEmailConfiguration", "smtp_password", "organization_id")
	someOrg       = uuid.MustParse("6f3f1a5e-0f7a-4f6d-9a1e-7c9a6b2d4e10")
	someOtherOrg  = uuid.MustParse("b2c7d9e1-3a45-4c8b-9f02-1d6e8a3b5c74")
	someCustomer  = uuid.MustParse("4d8e2f60-95b3-4a17-8c2d-5e0f7a1b3c96")
	someSecretVal = String("hunter2")
)

func TestEncryptDecrypt(t *testing.T) {
	t.Run("round trips a value", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		sealed, err := secretValue.Encrypt(someSecretVal, someCustomer, someOrg)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(sealed)).NotTo(ContainSubstring(string(someSecretVal)))
		g.Expect(secretValue.Decrypt(sealed, someCustomer, someOrg)).To(Equal([]byte(someSecretVal)))
	})

	t.Run("compresses a large value, which a column can no longer do for itself", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		plaintext := strings.Repeat("deployment log line\n", 1000)
		sealed, err := secretValue.Encrypt(String(plaintext), someCustomer, someOrg)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(len(sealed)).To(BeNumerically("<", len(plaintext)))
		g.Expect(secretValue.Decrypt(sealed, someCustomer, someOrg)).To(Equal([]byte(plaintext)))
	})

	t.Run("rejects a value moved to another column", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		sealed, err := smtpPassword.Encrypt(someSecretVal, someOrg)
		g.Expect(err).NotTo(HaveOccurred())
		_, err = smtpUsername.Decrypt(sealed, someOrg)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("rejects a value whose row was moved to another organization", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		sealed, err := secretValue.Encrypt(someSecretVal, someCustomer, someOrg)
		g.Expect(err).NotTo(HaveOccurred())
		_, err = secretValue.Decrypt(sealed, someCustomer, someOtherOrg)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("rejects a value whose row was handed to a customer", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		sealed, err := secretValue.Encrypt(someSecretVal, uuid.Nil, someOrg)
		g.Expect(err).NotTo(HaveOccurred())
		_, err = secretValue.Decrypt(sealed, someCustomer, someOrg)
		g.Expect(err).To(HaveOccurred())
	})

	t.Run("rejects a scope of the wrong length", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		_, err := secretValue.Encrypt(someSecretVal, someOrg)
		g.Expect(err).To(MatchError(ContainSubstring("Secret.value is bound to")))
	})
}

func TestDecryptScanned(t *testing.T) {
	t.Run("returns a plaintext marked value unchanged", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		g.Expect(decryptScanned(append([]byte{plaintextMarker}, someSecretVal...))).
			To(Equal([]byte(someSecretVal)))
	})

	t.Run("rejects an empty value", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		_, err := decryptScanned(nil)
		g.Expect(err).To(MatchError(ErrValueTooShort))
	})

	t.Run("rejects a value that is shorter than its announced authenticated data", func(t *testing.T) {
		g := NewWithT(t)
		useKeyring(t)
		_, err := decryptScanned([]byte{40, 1, 2, 3})
		g.Expect(err).To(MatchError(ErrValueTooShort))
	})
}

// TestScanPlan pins down that pgx routes an encrypted column into the decrypting Scan method rather
// than assigning the raw ciphertext to the underlying string or byte slice.
func TestScanPlan(t *testing.T) {
	useKeyring(t)
	scope := []uuid.UUID{someCustomer, someOrg}
	sealed, err := secretValue.Encrypt(someSecretVal, scope...)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	value := scanned(t, secretValue, scope, sealed)
	m := pgtype.NewMap()

	t.Run("String", func(t *testing.T) {
		g := NewWithT(t)
		var dst String
		plan := m.PlanScan(pgtype.ByteaOID, pgtype.BinaryFormatCode, &dst)
		g.Expect(plan).NotTo(BeNil())
		g.Expect(plan.Scan(value, &dst)).To(Succeed())
		g.Expect(dst).To(Equal(someSecretVal))
	})

	t.Run("nullable String", func(t *testing.T) {
		g := NewWithT(t)
		var dst *String
		plan := m.PlanScan(pgtype.ByteaOID, pgtype.BinaryFormatCode, &dst)
		g.Expect(plan).NotTo(BeNil())
		g.Expect(plan.Scan(value, &dst)).To(Succeed())
		g.Expect(dst).To(HaveValue(Equal(someSecretVal)))

		g.Expect(plan.Scan(nil, &dst)).To(Succeed())
		g.Expect(dst).To(BeNil())
	})

	t.Run("Bytes", func(t *testing.T) {
		g := NewWithT(t)
		var dst Bytes
		plan := m.PlanScan(pgtype.ByteaOID, pgtype.BinaryFormatCode, &dst)
		g.Expect(plan).NotTo(BeNil())
		g.Expect(plan.Scan(value, &dst)).To(Succeed())
		g.Expect(dst).To(Equal(Bytes(someSecretVal)))

		g.Expect(plan.Scan(nil, &dst)).To(Succeed())
		g.Expect(dst).To(BeNil())
	})
}
