package apitest_test

import (
	"net/http"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestAccessTokenLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	created := apitest.Expect[api.AccessTokenWithKey](vendor.Post(apitest.AccessTokens,
		api.CreateAccessTokenRequest{Label: new("full")}), http.StatusCreated)
	token := h.WithAccessToken(created.Key)
	g := NewWithT(t)

	g.Expect(apitest.QueryValue[bool](h, `SELECT last_used_at IS NULL FROM AccessToken WHERE id = $1`, created.ID)).
		To(BeTrue())
	token.Get(apitest.Applications).ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[bool](h, `SELECT last_used_at IS NULL FROM AccessToken WHERE id = $1`, created.ID)).
		To(BeFalse(), "a use of the token is recorded")

	token.Post(apitest.AccessTokens, api.CreateAccessTokenRequest{Label: new("minted")}).
		ExpectStatus(http.StatusForbidden)

	vendor.Invite(types.UserRoleAdmin, nil).Delete(apitest.Path(apitest.AccessTokens, created.ID))
	token.Get(apitest.Applications).ExpectStatus(http.StatusOK)

	vendor.Delete(apitest.Path(apitest.AccessTokens, created.ID)).ExpectStatus(http.StatusNoContent)
	token.Get(apitest.Applications).ExpectStatus(http.StatusUnauthorized)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM AccessToken`)).To(Equal(0))
}

func TestAccessTokenRoleCapsTheUserRole(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	created := apitest.Expect[api.AccessTokenWithKey](vendor.Post(apitest.AccessTokens,
		api.CreateAccessTokenRequest{UserRole: new(types.UserRoleReadOnly)}), http.StatusCreated)
	token := h.WithAccessToken(created.Key)

	token.Get(apitest.Applications).ExpectStatus(http.StatusOK)
	token.Post(apitest.Applications, api.CreateApplicationRequest{Name: "new", Type: types.DeploymentTypeDocker}).
		ExpectStatus(http.StatusForbidden)
	NewWithT(t).Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Application`)).To(Equal(0))
}
