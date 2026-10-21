package apitest

import (
	"net/http"
	"net/url"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

const Password = "Integration-Test-123!"

// User is a logged-in user whose requests carry their login token.
type User struct {
	*Client
	ID             uuid.UUID
	Email          string
	OrganizationID uuid.UUID
	// Token is the token the login returned.
	Token string
}

// RegisterVendor registers a user with an organization of its own, which makes them its admin.
func (h *Harness) RegisterVendor() *User {
	h.t.Helper()
	email := uniqueEmail("vendor")
	login := Expect[api.AuthLoginResponse](h.Anonymous().Post("/api/v1/auth/register", api.AuthRegistrationRequest{
		Name:             "Vendor",
		OrganizationName: "Vendor " + email,
		Email:            email,
		Password:         Password,
	}), http.StatusOK)
	return h.loggedIn(login.Token)
}

// Invite creates a user with role in the organization of u, or in the customer organization
// customerOrganizationID unless it is nil, and accepts the invitation in their name.
func (u *User) Invite(role types.UserRole, customerOrganizationID *uuid.UUID) *User {
	t := u.h.t
	t.Helper()
	created := Expect[api.CreateUserAccountResponse](u.Post(UserAccounts, api.CreateUserAccountRequest{
		Email:                  uniqueEmail(string(role)),
		Name:                   "Invited " + string(role),
		UserRole:               role,
		CustomerOrganizationID: customerOrganizationID,
	}), http.StatusOK)
	inviteURL, err := url.Parse(created.InviteURL)
	if err != nil {
		t.Fatalf("invite URL %q is invalid: %v", created.InviteURL, err)
	}
	token := inviteURL.Query().Get("jwt")
	if token == "" {
		t.Fatalf("invite URL %q has no jwt", created.InviteURL)
	}
	login := Expect[api.AuthLoginResponse](
		u.h.WithBearer(token).Post("/api/v1/auth/invite/accept", api.AuthAcceptInviteRequest{Password: Password}),
		http.StatusOK,
	)
	return u.h.loggedIn(login.Token)
}

// GrantFeature gives the organization of u a feature its plan does not include. The next request
// sees it, since the organization is loaded per request.
func (u *User) GrantFeature(feature types.Feature) {
	u.h.t.Helper()
	u.h.Exec(`UPDATE Organization SET features = array_append(features, $1) WHERE id = $2`,
		feature, u.OrganizationID)
}

// SetSubscriptionType moves the organization of u onto another plan without touching its features.
func (u *User) SetSubscriptionType(subscriptionType types.SubscriptionType) {
	u.h.t.Helper()
	u.h.Exec(`UPDATE Organization SET subscription_type = $1 WHERE id = $2`, subscriptionType, u.OrganizationID)
}

func (h *Harness) loggedIn(token string) *User {
	h.t.Helper()
	client := h.WithBearer(token)
	ctx := Expect[api.ContextResponse](client.Get("/api/v1/context"), http.StatusOK)
	return &User{
		Client:         client,
		ID:             ctx.User.ID,
		Email:          ctx.User.Email,
		OrganizationID: ctx.Organization.ID,
		Token:          token,
	}
}

func uniqueEmail(prefix string) string {
	return prefix + "-" + uuid.NewString() + "@distr.test"
}
