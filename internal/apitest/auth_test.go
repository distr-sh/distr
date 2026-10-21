package apitest_test

import (
	"net/http"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

const (
	authLogin         = "/api/v1/auth/login"
	authRegister      = "/api/v1/auth/register"
	authReset         = "/api/v1/auth/reset"
	authResetConfirm  = "/api/v1/auth/reset/confirm"
	authInviteAccept  = "/api/v1/auth/invite/accept"
	authSwitchContext = "/api/v1/auth/switch-context"
	contextPath       = "/api/v1/context"
	newPassword       = "New-Password-123!"
)

var mailedJWT = regexp.MustCompile(`jwt=([^"&\s<]+)`)

func mailedToken(t *testing.T, mail apitest.Mail) string {
	t.Helper()
	match := mailedJWT.FindStringSubmatch(mail.HTMLBody)
	if match == nil {
		t.Fatalf("mail %q contains no jwt link", mail.Subject)
	}
	token, err := url.QueryUnescape(match[1])
	if err != nil {
		t.Fatalf("jwt in mail %q is not URL-encoded: %v", mail.Subject, err)
	}
	return token
}

func requestPasswordReset(t *testing.T, h *apitest.Harness, email string) *apitest.Client {
	t.Helper()
	h.Anonymous().Post(authReset, api.AuthResetPasswordRequest{Email: email}).ExpectStatus(http.StatusNoContent)
	mails := h.Mails()
	return h.WithBearer(mailedToken(t, mails[len(mails)-1]))
}

func login(h *apitest.Harness, email, password string, mfaCode *string) *apitest.Response {
	return h.Anonymous().Post(authLogin, api.AuthLoginRequest{Email: email, Password: password, MFACode: mfaCode})
}

func TestLogin(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	anon := h.Anonymous()
	g := NewWithT(t)

	g.Expect(statuses(anon, []request{
		{"wrong password", http.MethodPost, authLogin, api.AuthLoginRequest{Email: vendor.Email, Password: "wrong"}},
		{"unknown email", http.MethodPost, authLogin, api.AuthLoginRequest{Email: "nobody@distr.test", Password: "x"}},
		{"invalid email", http.MethodPost, authLogin, api.AuthLoginRequest{Email: "nobody", Password: "x"}},
		{"malformed body", http.MethodPost, authLogin, "{"},
		{"empty body", http.MethodPost, authLogin, nil},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(apitest.QueryValue[*time.Time](h, `SELECT last_logged_in_at FROM UserAccount WHERE id = $1`,
		vendor.ID)).To(BeNil())

	response := apitest.Expect[api.AuthLoginResponse](login(h, vendor.Email, apitest.Password, nil), http.StatusOK)
	g.Expect(response.RequiresMFA).To(BeFalse())
	ctx := apitest.Expect[api.ContextResponse](h.WithBearer(response.Token).Get(contextPath), http.StatusOK)
	g.Expect(ctx.User.ID).To(Equal(vendor.ID))
	g.Expect(ctx.Organization.ID).To(Equal(vendor.OrganizationID))
	g.Expect(apitest.QueryValue[*time.Time](h, `SELECT last_logged_in_at FROM UserAccount WHERE id = $1`,
		vendor.ID)).NotTo(BeNil())
}

func TestRegistration(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	existing := h.RegisterVendor()
	anon := h.Anonymous()
	g := NewWithT(t)

	register := func(name, email, password string) request {
		return request{name, http.MethodPost, authRegister, api.AuthRegistrationRequest{
			Name: "Name", OrganizationName: "Organization", Email: email, Password: password,
		}}
	}
	g.Expect(statuses(anon, []request{
		register("duplicate email", existing.Email, apitest.Password),
		register("invalid email", "nobody", apitest.Password),
		register("weak password", "weak@distr.test", "short"),
		{"malformed body", http.MethodPost, authRegister, "{"},
	})).To(HaveEach(http.StatusBadRequest))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM UserAccount`)).To(Equal(1))

	email := "registered-" + uuid.NewString() + "@distr.test"
	response := apitest.Expect[api.AuthLoginResponse](anon.Post(authRegister, api.AuthRegistrationRequest{
		Name: "Registered", OrganizationName: "Registered Organization", Email: email, Password: apitest.Password,
	}), http.StatusOK)
	ctx := apitest.Expect[api.ContextResponse](h.WithBearer(response.Token).Get(contextPath), http.StatusOK)
	g.Expect(ctx.User.Email).To(Equal(email))
	g.Expect(ctx.User.UserRole).To(Equal(types.UserRoleAdmin))
	g.Expect(ctx.Organization.Name).To(Equal("Registered Organization"))
	g.Expect(ctx.Organization.ID).NotTo(Equal(existing.OrganizationID))
}

func TestPasswordReset(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	anon := h.Anonymous()
	g := NewWithT(t)

	anon.Post(authReset, api.AuthResetPasswordRequest{Email: "nobody@distr.test"}).ExpectStatus(http.StatusNoContent)
	g.Expect(h.Mails()).To(BeEmpty())
	anon.Post(authReset, api.AuthResetPasswordRequest{Email: "nobody"}).ExpectStatus(http.StatusBadRequest)

	reset := requestPasswordReset(t, h, vendor.Email)
	g.Expect(h.Mails()).To(HaveLen(1))
	g.Expect(h.Mails()[0].To).To(Equal([]string{vendor.Email}))
	g.Expect(h.Mails()[0].Subject).To(Equal("Password reset"))

	g.Expect(statuses(reset, []request{
		{"context", http.MethodGet, contextPath, nil},
		{"applications", http.MethodGet, apitest.Applications, nil},
		{"accept invite", http.MethodPost, authInviteAccept, api.AuthAcceptInviteRequest{Password: newPassword}},
	})).To(HaveEach(http.StatusUnauthorized))
	vendor.Post(authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: newPassword}).
		ExpectStatus(http.StatusUnauthorized)
	reset.Post(authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: "short"}).
		ExpectStatus(http.StatusBadRequest)
	login(h, vendor.Email, apitest.Password, nil).ExpectStatus(http.StatusOK)

	confirmed := apitest.Expect[api.AuthLoginResponse](
		reset.Post(authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: newPassword}), http.StatusOK)
	g.Expect(apitest.Expect[api.ContextResponse](h.WithBearer(confirmed.Token).Get(contextPath), http.StatusOK).
		User.ID).To(Equal(vendor.ID))
	login(h, vendor.Email, apitest.Password, nil).ExpectStatus(http.StatusBadRequest)
	login(h, vendor.Email, newPassword, nil).ExpectStatus(http.StatusOK)
}

func TestPasswordResetLinkCanOnlyBeUsedOnce(t *testing.T) {
	knownBug(t, "a reset token is a stateless JWT, so a reset link sets the password again until it expires")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()

	reset := requestPasswordReset(t, h, vendor.Email)
	reset.Post(authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: newPassword}).
		ExpectStatus(http.StatusOK)
	reset.Post(authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: "Another-Password-123!"}).
		ExpectStatus(http.StatusUnauthorized)
	login(h, vendor.Email, newPassword, nil).ExpectStatus(http.StatusOK)
}

func TestInvitation(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	g := NewWithT(t)

	email := "invited-" + uuid.NewString() + "@distr.test"
	created := apitest.Expect[api.CreateUserAccountResponse](vendor.Post(apitest.UserAccounts,
		api.CreateUserAccountRequest{Email: email, Name: "Invited", UserRole: types.UserRoleReadWrite}), http.StatusOK)
	g.Expect(created.User.Activated).To(BeFalse())
	g.Expect(h.Mails()).To(HaveLen(1))
	g.Expect(h.Mails()[0].To).To(Equal([]string{email}))

	inviteURL, err := url.Parse(created.InviteURL)
	g.Expect(err).NotTo(HaveOccurred())
	invite := h.WithBearer(inviteURL.Query().Get("jwt"))
	g.Expect(statuses(invite, []request{
		{"context", http.MethodGet, contextPath, nil},
		{"applications", http.MethodGet, apitest.Applications, nil},
		{"confirm reset", http.MethodPost, authResetConfirm, api.AuthResetPasswordConfirmRequest{Password: newPassword}},
	})).To(HaveEach(http.StatusUnauthorized))
	invite.Post(authInviteAccept, api.AuthAcceptInviteRequest{Password: "short"}).ExpectStatus(http.StatusBadRequest)
	login(h, email, apitest.Password, nil).ExpectStatus(http.StatusBadRequest)

	resendPath := apitest.Path(apitest.UserAccounts, created.User.ID) + "/invite"
	vendor.Post(resendPath, nil).ExpectStatus(http.StatusOK)
	g.Expect(h.Mails()).To(HaveLen(2))
	resent := h.WithBearer(mailedToken(t, h.Mails()[1]))

	accepted := apitest.Expect[api.AuthLoginResponse](resent.Post(authInviteAccept,
		api.AuthAcceptInviteRequest{Name: new("Renamed"), Password: apitest.Password}), http.StatusOK)
	ctx := apitest.Expect[api.ContextResponse](h.WithBearer(accepted.Token).Get(contextPath), http.StatusOK)
	g.Expect(ctx.User.ID).To(Equal(created.User.ID))
	g.Expect(ctx.User.Name).To(Equal("Renamed"))
	g.Expect(ctx.User.UserRole).To(Equal(types.UserRoleReadWrite))
	g.Expect(ctx.Organization.ID).To(Equal(vendor.OrganizationID))

	invite.Post(authInviteAccept, api.AuthAcceptInviteRequest{Password: newPassword}).
		ExpectStatus(http.StatusBadRequest)
	login(h, email, apitest.Password, nil).ExpectStatus(http.StatusOK)
	g.Expect(statuses(vendor.Client, []request{
		{"activated user", http.MethodPost, resendPath, nil},
		{"malformed id", http.MethodPost, apitest.UserAccounts + "/nope/invite", nil},
		{"unknown user", http.MethodPost, apitest.Path(apitest.UserAccounts, uuid.New()) + "/invite", nil},
	})).To(Equal(map[string]int{
		"activated user": http.StatusBadRequest,
		"malformed id":   http.StatusNotFound,
		"unknown user":   http.StatusNotFound,
	}))
}

func TestSwitchContext(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	member := h.RegisterVendor()
	outsider := h.RegisterVendor()
	g := NewWithT(t)

	vendor.Post(apitest.UserAccounts, api.CreateUserAccountRequest{
		Email: member.Email, Name: "Member", UserRole: types.UserRoleReadOnly,
	}).ExpectStatus(http.StatusOK)
	g.Expect(apitest.IDs(member.Get("/api/v1/organizations"), http.StatusOK)).
		To(ConsistOf(vendor.OrganizationID, member.OrganizationID))

	g.Expect(statuses(member.Client, []request{
		{
			"own organization", http.MethodPost, authSwitchContext,
			api.AuthSwitchContextRequest{OrganizationID: member.OrganizationID},
		},
		{"no organization", http.MethodPost, authSwitchContext, api.AuthSwitchContextRequest{}},
		{
			"unknown organization", http.MethodPost, authSwitchContext,
			api.AuthSwitchContextRequest{OrganizationID: uuid.New()},
		},
		{"malformed body", http.MethodPost, authSwitchContext, "{"},
	})).To(Equal(map[string]int{
		"own organization":     http.StatusNoContent,
		"no organization":      http.StatusBadRequest,
		"unknown organization": http.StatusBadRequest,
		"malformed body":       http.StatusBadRequest,
	}))
	outsider.Post(authSwitchContext, api.AuthSwitchContextRequest{OrganizationID: vendor.OrganizationID}).
		ExpectStatus(http.StatusBadRequest)

	switched := apitest.Expect[api.AuthLoginResponse](member.Post(authSwitchContext,
		api.AuthSwitchContextRequest{OrganizationID: vendor.OrganizationID}), http.StatusOK)
	client := h.WithBearer(switched.Token)
	ctx := apitest.Expect[api.ContextResponse](client.Get(contextPath), http.StatusOK)
	g.Expect(ctx.Organization.ID).To(Equal(vendor.OrganizationID))
	g.Expect(ctx.User.UserRole).To(Equal(types.UserRoleReadOnly))
	client.Post(apitest.Applications, api.CreateApplicationRequest{Name: "app", Type: types.DeploymentTypeDocker}).
		ExpectStatus(http.StatusForbidden)
	g.Expect(apitest.QueryValue[uuid.UUID](h, `SELECT last_used_organization_id FROM UserAccount WHERE id = $1`,
		member.ID)).To(Equal(vendor.OrganizationID))
}
