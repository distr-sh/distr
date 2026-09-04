package dbcrypto

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	. "github.com/onsi/gomega"
)

func testKey(fill byte) string {
	return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{fill}, MinKeyLen))
}

func mustKeyring(t *testing.T, spec string) *Keyring {
	t.Helper()
	keyring, err := ParseKeyring(spec)
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	return keyring
}

// useKeyring points the process wide keyring at a test keyring, so the [String] and [Bytes] scan
// paths can run without an initialized environment.
func useKeyring(t *testing.T, keyring *Keyring) {
	t.Helper()
	previous := keys
	keys = func() (*Keyring, error) { return keyring, nil }
	t.Cleanup(func() { keys = previous })
}

func TestParseKeyring(t *testing.T) {
	t.Run("a bare key is the active key with id 0", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		g.Expect(keyring.ActiveKeyID()).To(Equal(byte(0)))
		g.Expect(keyring.KeyIDs()).To(Equal([]byte{0}))
	})

	t.Run("the first entry is active regardless of its id", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, "7:"+testKey(1)+", 3:"+testKey(2))
		g.Expect(keyring.ActiveKeyID()).To(Equal(byte(7)))
		g.Expect(keyring.KeyIDs()).To(Equal([]byte{7, 3}))
	})

	t.Run("rejects malformed specs", func(t *testing.T) {
		for name, spec := range map[string]string{
			"empty":           "",
			"duplicate id":    "1:" + testKey(1) + ",1:" + testKey(2),
			"id out of range": "256:" + testKey(1),
			"non numeric id":  "primary:" + testKey(1),
			"not base64":      "1:not base64!",
			"too short":       "1:" + base64.StdEncoding.EncodeToString([]byte("short")),
		} {
			t.Run(name, func(t *testing.T) {
				_, err := ParseKeyring(spec)
				NewWithT(t).Expect(err).To(HaveOccurred())
			})
		}
	})
}

func TestKeyringEncryptDecrypt(t *testing.T) {
	t.Run("round trips a value", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		sealed, err := keyring.Encrypt([]byte("hunter2"))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(string(sealed)).NotTo(ContainSubstring("hunter2"))
		g.Expect(keyring.Decrypt(sealed)).To(Equal([]byte("hunter2")))
	})

	t.Run("seals the same plaintext differently every time", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		first, err := keyring.Encrypt([]byte("hunter2"))
		g.Expect(err).NotTo(HaveOccurred())
		second, err := keyring.Encrypt([]byte("hunter2"))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(first).NotTo(Equal(second))
	})

	t.Run("compresses a large value and still round trips it", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		plaintext := []byte(strings.Repeat("deployment log line\n", 1000))
		sealed, err := keyring.Encrypt(plaintext)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(sealed[2] & flagCompressed).NotTo(BeZero())
		g.Expect(len(sealed)).To(BeNumerically("<", len(plaintext)))
		g.Expect(keyring.Decrypt(sealed)).To(Equal(plaintext))
	})

	t.Run("leaves an incompressible value uncompressed", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		sealed, err := keyring.Encrypt([]byte("hunter2"))
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(sealed[2]).To(BeZero())
	})

	t.Run("returns a plaintext marked value unchanged", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		g.Expect(keyring.Decrypt(append([]byte{plaintextMarker}, "hunter2"...))).To(Equal([]byte("hunter2")))
	})

	t.Run("rejects an unrecognized format", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		_, err := keyring.Decrypt([]byte{99, 0, 0, 1, 2, 3})
		g.Expect(err).To(MatchError(ErrUnknownFormat))
	})

	t.Run("rejects a value sealed with a different key", func(t *testing.T) {
		g := NewWithT(t)
		sealed, err := mustKeyring(t, "1:"+testKey(1)).Encrypt([]byte("hunter2"))
		g.Expect(err).NotTo(HaveOccurred())

		_, err = mustKeyring(t, "1:"+testKey(2)).Decrypt(sealed)
		g.Expect(err).To(HaveOccurred())

		_, err = mustKeyring(t, "2:"+testKey(2)).Decrypt(sealed)
		g.Expect(err).To(MatchError(ErrUnknownKey))
	})
}

func TestKeyringRotation(t *testing.T) {
	g := NewWithT(t)
	old := mustKeyring(t, "1:"+testKey(1))
	sealed, err := old.Encrypt([]byte("hunter2"))
	g.Expect(err).NotTo(HaveOccurred())

	rotated := mustKeyring(t, "2:"+testKey(2)+",1:"+testKey(1))
	g.Expect(rotated.Decrypt(sealed)).To(Equal([]byte("hunter2")), "a value written before the rotation")

	resealed, err := rotated.Encrypt([]byte("hunter2"))
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(resealed[1]).To(Equal(byte(2)), "a write uses the new key")

	_, err = old.Decrypt(resealed)
	g.Expect(err).To(MatchError(ErrUnknownKey), "the old keyring cannot read the new key")
}

func TestKeyringHMAC(t *testing.T) {
	t.Run("is deterministic and does not leak the value", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		g.Expect(keyring.HMAC([]byte("token"))).To(Equal(keyring.HMAC([]byte("token"))))
		g.Expect(keyring.HMAC([]byte("token"))).NotTo(Equal(keyring.HMAC([]byte("other"))))
		g.Expect(string(keyring.HMAC([]byte("token")))).NotTo(ContainSubstring("token"))
	})

	t.Run("uses a key independent of the encryption key", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, testKey(1))
		encKey, err := keyring.active.enc.Encrypt(nil)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(encKey).NotTo(BeEmpty())
		g.Expect(keyring.active.mac).NotTo(Equal(bytes.Repeat([]byte{1}, MinKeyLen)))
		g.Expect(keyring.HMAC(nil)).NotTo(Equal(keyring.active.mac))
	})

	t.Run("differs per key so a rotation needs every candidate", func(t *testing.T) {
		g := NewWithT(t)
		keyring := mustKeyring(t, "2:"+testKey(2)+",1:"+testKey(1))
		all := keyring.HMACAll([]byte("token"))
		g.Expect(all).To(HaveLen(2))
		g.Expect(all[0]).To(Equal(keyring.HMAC([]byte("token"))), "the active key comes first")
		g.Expect(all[0]).NotTo(Equal(all[1]))
		g.Expect(all[1]).To(Equal(mustKeyring(t, "1:"+testKey(1)).HMAC([]byte("token"))))
	})
}

// TestScanPlan pins down that pgx routes an encrypted column into the decrypting Scan method rather
// than assigning the raw ciphertext to the underlying string or byte slice.
func TestScanPlan(t *testing.T) {
	keyring := mustKeyring(t, testKey(1))
	useKeyring(t, keyring)
	sealed, err := keyring.Encrypt([]byte("hunter2"))
	NewWithT(t).Expect(err).NotTo(HaveOccurred())
	m := pgtype.NewMap()

	t.Run("String", func(t *testing.T) {
		g := NewWithT(t)
		var dst String
		plan := m.PlanScan(pgtype.ByteaOID, pgtype.BinaryFormatCode, &dst)
		g.Expect(plan).NotTo(BeNil())
		g.Expect(plan.Scan(sealed, &dst)).To(Succeed())
		g.Expect(dst).To(Equal(String("hunter2")))
	})

	t.Run("nullable String", func(t *testing.T) {
		g := NewWithT(t)
		var dst *String
		plan := m.PlanScan(pgtype.ByteaOID, pgtype.BinaryFormatCode, &dst)
		g.Expect(plan).NotTo(BeNil())
		g.Expect(plan.Scan(sealed, &dst)).To(Succeed())
		g.Expect(dst).To(HaveValue(Equal(String("hunter2"))))

		g.Expect(plan.Scan(nil, &dst)).To(Succeed())
		g.Expect(dst).To(BeNil())
	})

	t.Run("Bytes", func(t *testing.T) {
		g := NewWithT(t)
		var dst Bytes
		plan := m.PlanScan(pgtype.ByteaOID, pgtype.BinaryFormatCode, &dst)
		g.Expect(plan).NotTo(BeNil())
		g.Expect(plan.Scan(sealed, &dst)).To(Succeed())
		g.Expect(dst).To(Equal(Bytes("hunter2")))

		g.Expect(plan.Scan(nil, &dst)).To(Succeed())
		g.Expect(dst).To(BeNil())
	})
}
