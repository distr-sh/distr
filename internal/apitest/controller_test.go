package apitest_test

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

const (
	controllerLogin      = "/api/v1/controller/login"
	controllerResources  = "/api/v1/controller/resources"
	controllerStatus     = "/api/v1/controller/status"
	controllerMetrics    = "/api/v1/controller/metrics"
	controllerLogs       = "/api/v1/controller/logs"
	controllerTargetLogs = "/api/v1/controller/deployment-target-logs"
)

type deployedTarget struct {
	target, deployment, revision uuid.UUID
}

// deployDockerTarget creates a Docker deployment target of vendor with one deployment of version.
func deployDockerTarget(vendor *apitest.User, version uuid.UUID, envFile string) deployedTarget {
	target := vendor.CreateDeploymentTarget("target", nil)
	deployment, revision := vendor.Deploy(api.DeploymentRequest{
		DeploymentTargetID:   target,
		ApplicationVersionID: version,
		EnvFileData:          []byte(envFile),
	})
	return deployedTarget{target: target, deployment: deployment, revision: revision}
}

func connectQuery(targetID, secret string) string {
	return "?" + url.Values{"targetId": {targetID}, "targetSecret": {secret}}.Encode()
}

func TestControllerConnect(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	target := vendor.CreateDeploymentTarget("target", nil)
	access := vendor.RequestAccess(target)
	anonymous := h.Anonymous()
	g := NewWithT(t)

	connect := anonymous.Get("/api/v1/connect" + connectQuery(target.String(), access.TargetSecret)).
		ExpectStatus(http.StatusOK)
	g.Expect(connect.Header.Get("Content-Type")).To(Equal("application/yaml"))
	g.Expect(string(connect.Body)).To(ContainSubstring("docker-controller"))
	preConnect := anonymous.Get("/api/v1/pre-connect" + connectQuery(target.String(), access.TargetSecret)).
		ExpectStatus(http.StatusOK)
	g.Expect(preConnect.Header.Get("Content-Type")).To(HavePrefix("text/plain"))
	g.Expect(string(preConnect.Body)).To(ContainSubstring("/api/v1/connect?targetId=" + target.String()))

	g.Expect(statuses(anonymous, []request{
		{"wrong secret", http.MethodGet, "/api/v1/connect" + connectQuery(target.String(), "wrong"), nil},
		{"missing secret", http.MethodGet, "/api/v1/connect?targetId=" + target.String(), nil},
		{"unknown target", http.MethodGet, "/api/v1/connect" + connectQuery(uuid.NewString(), "secret"), nil},
		{"malformed target", http.MethodGet, "/api/v1/connect" + connectQuery("not-a-uuid", "secret"), nil},
		{"no parameters", http.MethodGet, "/api/v1/connect", nil},
		{"pre-connect without parameters", http.MethodGet, "/api/v1/pre-connect", nil},
	})).To(Equal(map[string]int{
		"wrong secret":                   http.StatusUnauthorized,
		"missing secret":                 http.StatusUnauthorized,
		"unknown target":                 http.StatusUnauthorized,
		"malformed target":               http.StatusBadRequest,
		"no parameters":                  http.StatusBadRequest,
		"pre-connect without parameters": http.StatusBadRequest,
	}))
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT access_key_hash IS NULL AND pending_access_key_hash IS NOT NULL FROM DeploymentTarget WHERE id = $1`,
		target,
	)).To(BeTrue(), "connecting leaves the key pending until the controller logs in")
}

func TestControllerLogin(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	target := vendor.CreateDeploymentTarget("target", nil)
	access := vendor.RequestAccess(target)
	g := NewWithT(t)

	g.Expect(map[string]int{
		"no credentials":   h.Anonymous().Post(controllerLogin, nil).StatusCode,
		"malformed target": h.WithBasicAuth("not-a-uuid", access.TargetSecret).Post(controllerLogin, nil).StatusCode,
		"wrong secret":     h.WithBasicAuth(target.String(), "wrong").Post(controllerLogin, nil).StatusCode,
		"unknown target":   h.WithBasicAuth(uuid.NewString(), access.TargetSecret).Post(controllerLogin, nil).StatusCode,
	}).To(Equal(map[string]int{
		"no credentials":   http.StatusUnauthorized,
		"malformed target": http.StatusBadRequest,
		"wrong secret":     http.StatusUnauthorized,
		"unknown target":   http.StatusUnauthorized,
	}))

	h.LoginController(access)
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT access_key_hash IS NOT NULL AND pending_access_key_hash IS NULL FROM DeploymentTarget WHERE id = $1`,
		target,
	)).To(BeTrue(), "logging in promotes the pending key")

	legacy := apitest.Expect[api.AuthLoginResponse](
		h.WithBasicAuth(target.String(), access.TargetSecret).Post("/api/v1/agent/login", nil), http.StatusOK)
	h.WithBearer(legacy.Token).Get("/api/v1/agent/resources").ExpectStatus(http.StatusOK)
}

func TestControllerAccessKeyRotation(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	target := vendor.CreateDeploymentTarget("target", nil)
	first := vendor.RequestAccess(target)
	firstController := h.LoginController(first)
	login := func(access api.DeploymentTargetAccessTokenResponse) int {
		return h.WithBasicAuth(target.String(), access.TargetSecret).Post(controllerLogin, nil).StatusCode
	}
	g := NewWithT(t)

	second := vendor.RequestAccess(target)
	g.Expect(login(first)).To(Equal(http.StatusOK), "the current key stays valid while another one is pending")
	g.Expect(login(second)).To(Equal(http.StatusOK))
	g.Expect(login(first)).To(Equal(http.StatusUnauthorized), "the pending key replaced the current one")
	firstController.Get(controllerResources).
		ExpectStatus(http.StatusOK)
}

func TestControllerPendingAccessKeyCanBeWithdrawn(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	target := vendor.CreateDeploymentTarget("target", nil)
	current := vendor.RequestAccess(target)
	h.LoginController(current)

	withdrawn := vendor.RequestAccess(target)
	vendor.Delete(apitest.Path(apitest.DeploymentTargets, target) + "/access-request").
		ExpectStatus(http.StatusNoContent)
	g := NewWithT(t)
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT pending_access_key_hash IS NULL FROM DeploymentTarget WHERE id = $1`, target)).To(BeTrue())
	g.Expect(h.WithBasicAuth(target.String(), withdrawn.TargetSecret).Post(controllerLogin, nil).StatusCode).
		To(Equal(http.StatusUnauthorized))
	g.Expect(h.WithBasicAuth(target.String(), current.TargetSecret).Post(controllerLogin, nil).StatusCode).
		To(Equal(http.StatusOK))
}

func TestControllerRoutesRequireAControllerToken(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	target := vendor.CreateDeploymentTarget("target", nil)
	controller := h.LoginController(vendor.RequestAccess(target))
	accessToken := apitest.Expect[api.AccessTokenWithKey](
		vendor.Post(apitest.AccessTokens, api.CreateAccessTokenRequest{}), http.StatusCreated)

	controllerRoutes := []request{
		{"resources", http.MethodGet, controllerResources, nil},
		{"legacy resources", http.MethodGet, "/api/v1/agent/resources", nil},
		{"manifest", http.MethodGet, "/api/v1/controller/manifest", nil},
		{"status", http.MethodPost, controllerStatus, api.ControllerDeploymentStatus{}},
		{"metrics", http.MethodPost, controllerMetrics, api.ControllerDeploymentTargetMetricsRequest{}},
		{
			"deployment metrics", http.MethodPost, "/api/v1/controller/deployments/" + uuid.NewString() + "/metrics",
			api.ControllerDeploymentResourceMetricsRequest{},
		},
		{"logs", http.MethodPut, controllerLogs, []api.DeploymentLogRecord{}},
		{"deployment target logs", http.MethodPut, controllerTargetLogs, []api.DeploymentTargetLogRecordRequest{}},
	}
	g := NewWithT(t)
	for name, client := range map[string]*apitest.Client{
		"anonymous":    h.Anonymous(),
		"user":         vendor.Client,
		"access token": h.WithAccessToken(accessToken.Key),
	} {
		g.Expect(statuses(client, controllerRoutes)).
			To(Equal(allWithStatus(controllerRoutes, http.StatusUnauthorized)), name)
	}

	userRoutes := []request{
		{"context", http.MethodGet, "/api/v1/context", nil},
		{"applications", http.MethodGet, apitest.Applications, nil},
		{"deployment targets", http.MethodGet, apitest.DeploymentTargets, nil},
	}
	g.Expect(statuses(controller, userRoutes)).To(Equal(allWithStatus(userRoutes, http.StatusUnauthorized)))

	vendor.Delete(apitest.Path(apitest.DeploymentTargets, target)).ExpectStatus(http.StatusNoContent)
	g.Expect(controller.Get(controllerResources).StatusCode).
		To(Equal(http.StatusUnauthorized), "the token of a deleted deployment target is useless")
}

func TestControllerReceivesItsDeployments(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	vendor.CreateSecret("API_KEY", "s3cret", nil)
	application := vendor.CreateApplication("app")
	version := vendor.CreateApplicationVersion(application, "1.0.0")
	own := deployDockerTarget(vendor, version, "API_KEY={{ .Secrets.API_KEY }}\n")
	other := deployDockerTarget(vendor, version, "")
	controller := h.LoginController(vendor.RequestAccess(own.target))
	g := NewWithT(t)

	resources := apitest.Expect[api.ControllerResource](controller.Get(controllerResources), http.StatusOK)
	g.Expect(resources.Version.Name).To(Equal("snapshot"))
	g.Expect(resources.Deployments).To(HaveLen(1))
	deployment := resources.Deployments[0]
	g.Expect(deployment.ID).To(Equal(own.deployment))
	g.Expect(deployment.RevisionID).To(Equal(own.revision))
	g.Expect(string(deployment.EnvFile)).To(Equal("API_KEY=s3cret\n"))
	g.Expect(string(deployment.ComposeFile)).To(ContainSubstring("name: distr-" + own.deployment.String()[:8]))
	g.Expect(string(deployment.ComposeFile)).To(ContainSubstring("image: nginx:1.29"))
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT reported_controller_version_id IS NOT NULL FROM DeploymentTarget WHERE id = $1`, own.target,
	)).To(BeTrue(), "the user agent reports the controller version")

	legacy := apitest.Expect[api.ControllerResource](controller.Get("/api/v1/agent/resources"), http.StatusOK)
	g.Expect(legacy.Deployments).To(HaveLen(1))
	manifest := controller.Get("/api/v1/controller/manifest").ExpectStatus(http.StatusOK)
	g.Expect(manifest.Header.Get("Content-Type")).To(Equal("application/yaml"))

	_, redeployed := vendor.Deploy(api.DeploymentRequest{
		DeploymentID:         &own.deployment,
		DeploymentTargetID:   own.target,
		ApplicationVersionID: version,
	})
	resources = apitest.Expect[api.ControllerResource](controller.Get(controllerResources), http.StatusOK)
	g.Expect(resources.Deployments).To(HaveLen(1))
	g.Expect(resources.Deployments[0].RevisionID).To(Equal(redeployed))
	g.Expect(resources.Deployments[0].RevisionID).NotTo(Equal(own.revision))

	vendor.Delete(apitest.Path(apitest.Deployments, own.deployment)).ExpectStatus(http.StatusOK)
	resources = apitest.Expect[api.ControllerResource](controller.Get(controllerResources), http.StatusOK)
	g.Expect(resources.Deployments).To(BeEmpty())

	otherResources := apitest.Expect[api.ControllerResource](
		h.LoginController(vendor.RequestAccess(other.target)).Get(controllerResources), http.StatusOK)
	g.Expect(otherResources.Deployments).To(HaveLen(1))
	g.Expect(otherResources.Deployments[0].ID).To(Equal(other.deployment))
}

func TestControllerReportsDeploymentStatus(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	own := deployDockerTarget(vendor, version, "")
	other := deployDockerTarget(vendor, version, "")
	controller := h.LoginController(vendor.RequestAccess(own.target))
	status := func(revision uuid.UUID, statusType types.DeploymentStatusType) api.ControllerDeploymentStatus {
		return api.ControllerDeploymentStatus{RevisionID: revision, Type: statusType, Message: string(statusType)}
	}
	revisionState := func(revision uuid.UUID) string {
		return apitest.QueryValue[string](h,
			`SELECT coalesce(status_type::TEXT, '') || '/' || coalesce(status_message, '') FROM DeploymentRevision
			WHERE id = $1`,
			revision,
		)
	}
	currentRevision := func(deployment uuid.UUID) *uuid.UUID {
		return apitest.QueryValue[*uuid.UUID](h,
			`SELECT current_deployment_revision_id FROM Deployment WHERE id = $1`, deployment)
	}
	controller.Get(controllerResources).ExpectStatus(http.StatusOK)
	g := NewWithT(t)

	controller.Post(controllerStatus, status(own.revision, types.DeploymentStatusTypeProgressing)).
		ExpectStatus(http.StatusOK)
	g.Expect(revisionState(own.revision)).To(Equal("progressing/progressing"))
	g.Expect(currentRevision(own.deployment)).To(BeNil(), "a progressing revision is not applied yet")

	controller.Post(controllerStatus, status(own.revision, types.DeploymentStatusTypeHealthy)).
		ExpectStatus(http.StatusOK)
	g.Expect(revisionState(own.revision)).To(Equal("healthy/healthy"))
	g.Expect(currentRevision(own.deployment)).To(HaveValue(Equal(own.revision)))

	g.Expect(statuses(controller, []request{
		{"unknown revision", http.MethodPost, controllerStatus, status(uuid.New(), types.DeploymentStatusTypeHealthy)},
		{
			"revision of another target", http.MethodPost, controllerStatus,
			status(other.revision, types.DeploymentStatusTypeHealthy),
		},
		{
			"unknown status type", http.MethodPost, controllerStatus,
			map[string]string{"revisionId": own.revision.String(), "type": "bogus", "message": "bogus"},
		},
		{"malformed body", http.MethodPost, controllerStatus, "not an object"},
		{"no body", http.MethodPost, controllerStatus, nil},
	})).To(Equal(map[string]int{
		"unknown revision":           http.StatusBadRequest,
		"revision of another target": http.StatusBadRequest,
		"unknown status type":        http.StatusBadRequest,
		"malformed body":             http.StatusBadRequest,
		"no body":                    http.StatusBadRequest,
	}))
	g.Expect(revisionState(own.revision)).To(Equal("healthy/healthy"))
	g.Expect(revisionState(other.revision)).To(Equal("/"))
	g.Expect(currentRevision(other.deployment)).To(BeNil())
}

func TestControllerReportsStatusRightAfterAVersionChange(t *testing.T) {
	knownBug(t, "db.UpdateDeploymentTargetReportedControllerVersionID replaces the deployment target "+
		"of the request with one without deployments, so the first request reporting a new controller version "+
		"answers 400 for a status and 404 for deployment metrics")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	own := deployDockerTarget(vendor, version, "")
	controller := h.LoginController(vendor.RequestAccess(own.target))

	controller.Post(controllerStatus, api.ControllerDeploymentStatus{
		RevisionID: own.revision, Type: types.DeploymentStatusTypeHealthy, Message: "healthy",
	}).ExpectStatus(http.StatusOK)
}

func TestControllerReportsMetrics(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	own := deployDockerTarget(vendor, version, "")
	other := deployDockerTarget(vendor, version, "")
	controller := h.LoginController(vendor.RequestAccess(own.target))
	deploymentMetrics := func(deployment string) string {
		return "/api/v1/controller/deployments/" + deployment + "/metrics"
	}
	resourceMetrics := api.ControllerDeploymentResourceMetricsRequest{
		Resources: []api.DeploymentResourceMetric{{Resource: "app"}},
	}
	controller.Get(controllerResources).ExpectStatus(http.StatusOK)
	g := NewWithT(t)

	vendor.Get(apitest.Path(apitest.Deployments, own.deployment) + "/metrics").ExpectStatus(http.StatusNoContent)
	controller.Post(controllerMetrics, api.ControllerDeploymentTargetMetricsRequest{
		CPUCoresMillis: 2000, CPUUsage: 0.5, MemoryBytes: 1 << 30, MemoryUsage: 0.25,
	}).ExpectStatus(http.StatusOK)
	controller.Post(deploymentMetrics(own.deployment.String()), resourceMetrics).ExpectStatus(http.StatusOK)

	g.Expect(apitest.QueryValue[int](h,
		`SELECT count(*) FROM DeploymentTargetMetrics WHERE deployment_target_id = $1`, own.target)).To(Equal(1))
	reported := apitest.Expect[api.DeploymentResourceMetrics](
		vendor.Get(apitest.Path(apitest.Deployments, own.deployment)+"/metrics"), http.StatusOK)
	g.Expect(reported.Resources).To(HaveLen(1))
	g.Expect(reported.Resources[0].Resource).To(Equal("app"))

	g.Expect(statuses(controller, []request{
		{"malformed target metrics", http.MethodPost, controllerMetrics, "not an object"},
		{"unknown deployment", http.MethodPost, deploymentMetrics(uuid.NewString()), resourceMetrics},
		{"deployment of another target", http.MethodPost, deploymentMetrics(other.deployment.String()), resourceMetrics},
		{"malformed deployment", http.MethodPost, deploymentMetrics("not-a-uuid"), resourceMetrics},
		{"malformed deployment metrics", http.MethodPost, deploymentMetrics(own.deployment.String()), "not an object"},
	})).To(Equal(map[string]int{
		"malformed target metrics":     http.StatusBadRequest,
		"unknown deployment":           http.StatusNotFound,
		"deployment of another target": http.StatusNotFound,
		"malformed deployment":         http.StatusBadRequest,
		"malformed deployment metrics": http.StatusBadRequest,
	}))
	vendor.Get(apitest.Path(apitest.Deployments, other.deployment) + "/metrics").ExpectStatus(http.StatusNoContent)
}

func TestControllerReportsLogs(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	own := deployDockerTarget(vendor, version, "")
	other := deployDockerTarget(vendor, version, "")
	controller := h.LoginController(vendor.RequestAccess(own.target))
	now := time.Now().UTC()
	record := func(target deployedTarget, body string) api.DeploymentLogRecord {
		return api.DeploymentLogRecord{
			DeploymentID:         target.deployment,
			DeploymentRevisionID: target.revision,
			Resource:             "app",
			Timestamp:            now,
			Severity:             "info",
			Body:                 body,
		}
	}
	bodies := func(path string) []string {
		records := apitest.Expect[[]api.DeploymentLogRecord](vendor.Get(path), http.StatusOK)
		result := make([]string, len(records))
		for i, r := range records {
			result[i] = r.Body
		}
		return result
	}
	g := NewWithT(t)

	controller.Put(controllerLogs, []api.DeploymentLogRecord{
		record(own, "hello"),
		record(own, "null\x00byte"),
		record(own, "\x00"),
		record(other, "smuggled"),
	}).ExpectStatus(http.StatusNoContent)
	g.Expect(bodies(apitest.Path(apitest.Deployments, own.deployment) + "/logs?resource=app")).
		To(ConsistOf("hello", "nullbyte"))
	g.Expect(bodies(apitest.Path(apitest.Deployments, other.deployment)+"/logs?resource=app")).
		To(BeEmpty(), "a controller cannot write the logs of another deployment target")

	controller.Put(controllerTargetLogs, []api.DeploymentTargetLogRecordRequest{
		{Timestamp: now, Severity: "info", Body: "controller started"},
	}).ExpectStatus(http.StatusNoContent)
	targetLogs := apitest.Expect[[]api.DeploymentTargetLogRecord](
		vendor.Get(apitest.Path(apitest.DeploymentTargets, own.target)+"/logs"), http.StatusOK)
	g.Expect(targetLogs).To(HaveLen(1))
	g.Expect(targetLogs[0].Body).To(Equal("controller started"))

	g.Expect(statuses(controller, []request{
		{"malformed logs", http.MethodPut, controllerLogs, "not a list"},
		{"malformed target logs", http.MethodPut, controllerTargetLogs, "not a list"},
	})).To(Equal(map[string]int{
		"malformed logs":        http.StatusBadRequest,
		"malformed target logs": http.StatusBadRequest,
	}))
	g.Expect(statuses(vendor.Client, []request{
		{"logs without resource", http.MethodGet, apitest.Path(apitest.Deployments, own.deployment) + "/logs", nil},
	})).To(Equal(map[string]int{"logs without resource": http.StatusBadRequest}))
}
