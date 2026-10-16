package apitest

import (
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/google/uuid"
)

const (
	Deployments = "/api/v1/deployments"
	// ControllerUserAgent is what a controller of the version every test database knows sends.
	ControllerUserAgent = "DistrControllerClient/snapshot"
)

// RequestAccess issues a new secret for the deployment target targetID, which becomes its pending
// access key.
func (u *User) RequestAccess(targetID uuid.UUID) api.DeploymentTargetAccessTokenResponse {
	u.h.t.Helper()
	return Expect[api.DeploymentTargetAccessTokenResponse](
		u.Post(Path(DeploymentTargets, targetID)+"/access-request", nil),
		http.StatusOK,
	)
}

// LoginController logs in as the controller of the deployment target access was issued for and
// returns a client sending its token and user agent.
func (h *Harness) LoginController(access api.DeploymentTargetAccessTokenResponse) *Client {
	h.t.Helper()
	return h.WithBearer(h.ControllerToken(access)).WithHeader("User-Agent", ControllerUserAgent)
}

// ControllerToken logs in as the controller of the deployment target access was issued for and
// returns its token.
func (h *Harness) ControllerToken(access api.DeploymentTargetAccessTokenResponse) string {
	h.t.Helper()
	return Expect[api.AuthLoginResponse](
		h.WithBasicAuth(access.TargetID.String(), access.TargetSecret).Post("/api/v1/controller/login", nil),
		http.StatusOK,
	).Token
}

// Deploy creates or updates a deployment and returns its ID and the ID of the revision the request
// created.
func (u *User) Deploy(request api.DeploymentRequest) (deploymentID, revisionID uuid.UUID) {
	u.h.t.Helper()
	u.Put(Deployments, request).ExpectStatus(http.StatusNoContent)
	if request.DeploymentID != nil {
		deploymentID = *request.DeploymentID
	} else {
		deploymentID = QueryValue[uuid.UUID](u.h,
			`SELECT id FROM Deployment WHERE deployment_target_id = $1 ORDER BY created_at DESC LIMIT 1`,
			request.DeploymentTargetID,
		)
	}
	revisionID = QueryValue[uuid.UUID](u.h,
		`SELECT id FROM DeploymentRevision WHERE deployment_id = $1 ORDER BY created_at DESC LIMIT 1`,
		deploymentID,
	)
	return deploymentID, revisionID
}
