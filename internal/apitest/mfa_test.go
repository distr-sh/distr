package apitest_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	. "github.com/onsi/gomega"
	"github.com/pquerna/otp/totp"
)

const (
	mfaSetup               = "/api/v1/settings/mfa/setup"
	mfaEnable              = "/api/v1/settings/mfa/enable"
	mfaDisable             = "/api/v1/settings/mfa/disable"
	mfaRecoveryCodesStatus = "/api/v1/settings/mfa/recovery-codes/status"
	mfaRegenerate          = "/api/v1/settings/mfa/recovery-codes/regenerate"
)

func totpCode(t *testing.T, secret string) string {
	t.Helper()
	code, err := totp.GenerateCode(secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate TOTP code: %v", err)
	}
	return code
}

// enableMFA enrolls u in MFA and returns the TOTP secret and the recovery codes.
func enableMFA(t *testing.T, u *apitest.User) (string, []string) {
	t.Helper()
	setup := apitest.Expect[api.SetupMFAResponse](u.Post(mfaSetup, nil), http.StatusOK)
	enabled := apitest.Expect[api.EnableMFAResponse](
		u.Post(mfaEnable, api.EnableMFARequest{Code: totpCode(t, setup.Secret)}), http.StatusOK)
	return setup.Secret, enabled.RecoveryCodes
}

func TestMFAEnrollment(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		{"status", http.MethodGet, mfaRecoveryCodesStatus, nil},
		{"enable", http.MethodPost, mfaEnable, api.EnableMFARequest{Code: "123456"}},
		{"disable", http.MethodPost, mfaDisable, api.DisableMFARequest{Password: apitest.Password}},
	})).To(HaveEach(http.StatusBadRequest))

	access := apitest.Expect[api.AccessTokenWithKey](
		vendor.Post(apitest.AccessTokens, api.CreateAccessTokenRequest{}), http.StatusCreated)
	h.WithAccessToken(access.Key).Post(mfaSetup, nil).ExpectStatus(http.StatusForbidden)

	setup := apitest.Expect[api.SetupMFAResponse](vendor.Post(mfaSetup, nil), http.StatusOK)
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT mfa_secret IS NULL AND mfa_secret_enc IS NOT NULL
			AND position(convert_to($2, 'UTF8') IN mfa_secret_enc) = 0
		FROM UserAccount WHERE id = $1`,
		vendor.ID, setup.Secret,
	)).To(BeTrue())
	g.Expect(apitest.Expect[api.AuthLoginResponse](login(h, vendor.Email, apitest.Password, nil), http.StatusOK).
		RequiresMFA).To(BeFalse())

	vendor.Post(mfaEnable, api.EnableMFARequest{Code: "000000"}).ExpectStatus(http.StatusBadRequest)
	g.Expect(apitest.QueryValue[bool](h, `SELECT mfa_enabled FROM UserAccount WHERE id = $1`, vendor.ID)).
		To(BeFalse())

	code := totpCode(t, setup.Secret)
	enabled := apitest.Expect[api.EnableMFAResponse](vendor.Post(mfaEnable, api.EnableMFARequest{Code: code}),
		http.StatusOK)
	g.Expect(enabled.RecoveryCodes).To(HaveLen(10))
	g.Expect(apitest.Expect[api.MFARecoveryCodesStatusResponse](vendor.Get(mfaRecoveryCodesStatus), http.StatusOK).
		RemainingCodes).To(Equal(10))
	g.Expect(apitest.QueryValue[int](h,
		`SELECT count(*) FROM UserAccount_MFARecoveryCode
		WHERE user_account_id = $1 AND position(convert_to($2, 'UTF8') IN code_hash) = 0`,
		vendor.ID, enabled.RecoveryCodes[0],
	)).To(Equal(10))

	g.Expect(statuses(vendor.Client, []request{
		{"setup again", http.MethodPost, mfaSetup, nil},
		{"enable again", http.MethodPost, mfaEnable, api.EnableMFARequest{Code: code}},
	})).To(HaveEach(http.StatusBadRequest))
}

func TestMFALogin(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	secret, recoveryCodes := enableMFA(t, vendor)
	g := NewWithT(t)

	required := apitest.Expect[api.AuthLoginResponse](login(h, vendor.Email, apitest.Password, nil), http.StatusOK)
	g.Expect(required.RequiresMFA).To(BeTrue())
	g.Expect(required.Token).To(BeEmpty())
	login(h, vendor.Email, apitest.Password, new("000000")).ExpectStatus(http.StatusUnauthorized)
	login(h, vendor.Email, "wrong", new(totpCode(t, secret))).ExpectStatus(http.StatusBadRequest)

	loggedIn := apitest.Expect[api.AuthLoginResponse](
		login(h, vendor.Email, apitest.Password, new(totpCode(t, secret))), http.StatusOK)
	h.WithBearer(loggedIn.Token).Get(contextPath).ExpectStatus(http.StatusOK)

	login(h, vendor.Email, apitest.Password, &recoveryCodes[0]).ExpectStatus(http.StatusOK)
	login(h, vendor.Email, apitest.Password, &recoveryCodes[0]).ExpectStatus(http.StatusUnauthorized)
	g.Expect(apitest.Expect[api.MFARecoveryCodesStatusResponse](vendor.Get(mfaRecoveryCodesStatus), http.StatusOK).
		RemainingCodes).To(Equal(9))

	vendor.Post(mfaRegenerate, api.RegenerateMFARecoveryCodesRequest{Password: "wrong"}).
		ExpectStatus(http.StatusUnauthorized)
	regenerated := apitest.Expect[api.RegenerateMFARecoveryCodesResponse](
		vendor.Post(mfaRegenerate, api.RegenerateMFARecoveryCodesRequest{Password: apitest.Password}), http.StatusOK)
	g.Expect(regenerated.RecoveryCodes).To(HaveLen(10))
	g.Expect(regenerated.RecoveryCodes).NotTo(ContainElements(recoveryCodes))
	login(h, vendor.Email, apitest.Password, &recoveryCodes[1]).ExpectStatus(http.StatusUnauthorized)
	login(h, vendor.Email, apitest.Password, &regenerated.RecoveryCodes[0]).ExpectStatus(http.StatusOK)
}

func TestMFAProtectsPasswordReset(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	secret, _ := enableMFA(t, vendor)
	reset := requestPasswordReset(t, h, vendor.Email)
	g := NewWithT(t)

	required := apitest.Expect[api.AuthLoginResponse](
		reset.Post(authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: newPassword}), http.StatusOK)
	g.Expect(required.RequiresMFA).To(BeTrue())
	g.Expect(required.Token).To(BeEmpty())
	reset.Post(authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: newPassword, MFACode: new("000000")}).
		ExpectStatus(http.StatusBadRequest)
	login(h, vendor.Email, newPassword, new(totpCode(t, secret))).ExpectStatus(http.StatusBadRequest)

	confirmed := apitest.Expect[api.AuthLoginResponse](reset.Post(authResetConfirm,
		api.AuthResetPasswordConfirmRequest{Password: newPassword, MFACode: new(totpCode(t, secret))}), http.StatusOK)
	g.Expect(confirmed.Token).NotTo(BeEmpty())
	login(h, vendor.Email, newPassword, new(totpCode(t, secret))).ExpectStatus(http.StatusOK)
}

func TestMFADisable(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	enableMFA(t, vendor)
	g := NewWithT(t)

	vendor.Post(mfaDisable, api.DisableMFARequest{Password: "wrong"}).ExpectStatus(http.StatusUnauthorized)
	g.Expect(apitest.Expect[api.AuthLoginResponse](login(h, vendor.Email, apitest.Password, nil), http.StatusOK).
		RequiresMFA).To(BeTrue())

	vendor.Post(mfaDisable, api.DisableMFARequest{Password: apitest.Password}).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.Expect[api.AuthLoginResponse](login(h, vendor.Email, apitest.Password, nil), http.StatusOK).
		Token).NotTo(BeEmpty())
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT NOT mfa_enabled AND mfa_secret_enc IS NULL
			AND NOT EXISTS (SELECT 1 FROM UserAccount_MFARecoveryCode WHERE user_account_id = $1)
		FROM UserAccount WHERE id = $1`,
		vendor.ID,
	)).To(BeTrue())
	vendor.Post(mfaDisable, api.DisableMFARequest{Password: apitest.Password}).ExpectStatus(http.StatusBadRequest)
}
