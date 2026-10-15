package apitest_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

const composeFile = "services:\n  app:\n    image: nginx:1.29\n"

func versionForm(version api.CreateApplicationVersionRequest, files map[string]string) apitest.Multipart {
	return apitest.Multipart{Fields: map[string]any{"applicationversion": version}, Files: files}
}

func dockerTarget(settings api.DeploymentTargetSettings) api.CreateDeploymentTargetRequest {
	return api.CreateDeploymentTargetRequest{DeploymentTargetSettings: settings, Type: types.DeploymentTypeDocker}
}

func kubernetesTarget(settings api.DeploymentTargetSettings, namespace *string) api.CreateDeploymentTargetRequest {
	return api.CreateDeploymentTargetRequest{
		DeploymentTargetSettings: settings,
		Type:                     types.DeploymentTypeKubernetes,
		Namespace:                namespace,
		Scope:                    new(types.DeploymentTargetScopeNamespace),
	}
}

func TestApplicationErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	otherVersion := vendor.CreateApplicationVersion(vendor.CreateApplication("other"), "1.0.0")
	appPath := apitest.Path(apitest.Applications, app)
	unknownPath := apitest.Path(apitest.Applications, uuid.New())
	g := NewWithT(t)

	create := func(name string, body any) request {
		return request{name, http.MethodPost, apitest.Applications, body}
	}
	g.Expect(statuses(vendor.Client, []request{
		create("create without name", api.CreateApplicationRequest{Type: types.DeploymentTypeDocker}),
		create("create without type", api.CreateApplicationRequest{Name: "x"}),
		create("create with unknown type", api.CreateApplicationRequest{Name: "x", Type: "foo"}),
		create("create with unknown versioning strategy",
			api.CreateApplicationRequest{Name: "x", Type: types.DeploymentTypeDocker, VersioningStrategy: "foo"}),
		create("create with automatic updates", api.CreateApplicationRequest{
			Name: "x", Type: types.DeploymentTypeDocker, AllowAutomaticUpdates: true,
		}),
		create("create with malformed body", "{"),
		{"get malformed id", http.MethodGet, apitest.Applications + "/nope", nil},
		{"get unknown", http.MethodGet, unknownPath, nil},
		{"put without name", http.MethodPut, appPath, api.UpdateApplicationRequest{}},
		{
			"put with unknown versioning strategy", http.MethodPut, appPath,
			api.UpdateApplicationRequest{Name: "app", VersioningStrategy: "foo"},
		},
		{
			"put with automatic updates", http.MethodPut, appPath,
			api.UpdateApplicationRequest{Name: "app", AllowAutomaticUpdates: new(true)},
		},
		{"put with malformed body", http.MethodPut, appPath, "{"},
		{"put malformed id", http.MethodPut, apitest.Applications + "/nope", api.UpdateApplicationRequest{Name: "x"}},
		{"put unknown", http.MethodPut, unknownPath, api.UpdateApplicationRequest{Name: "x"}},
		{"patch to empty name", http.MethodPatch, appPath, api.PatchApplicationRequest{Name: new("")}},
		{"patch with malformed body", http.MethodPatch, appPath, "{"},
		{
			"patch unknown version", http.MethodPatch, appPath,
			api.PatchApplicationRequest{Versions: []api.PatchApplicationVersionRequest{{ID: uuid.New()}}},
		},
		{
			"patch version of another application", http.MethodPatch, appPath,
			api.PatchApplicationRequest{Versions: []api.PatchApplicationVersionRequest{{ID: otherVersion}}},
		},
		{"delete malformed id", http.MethodDelete, apitest.Applications + "/nope", nil},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"create without name":                     http.StatusBadRequest,
		"create without type":                     http.StatusBadRequest,
		"create with unknown type":                http.StatusBadRequest,
		"create with unknown versioning strategy": http.StatusBadRequest,
		"create with automatic updates":           http.StatusBadRequest,
		"create with malformed body":              http.StatusBadRequest,
		"get malformed id":                        http.StatusNotFound,
		"get unknown":                             http.StatusNotFound,
		"put without name":                        http.StatusBadRequest,
		"put with unknown versioning strategy":    http.StatusBadRequest,
		"put with automatic updates":              http.StatusBadRequest,
		"put with malformed body":                 http.StatusBadRequest,
		"put malformed id":                        http.StatusNotFound,
		"put unknown":                             http.StatusNotFound,
		"patch to empty name":                     http.StatusBadRequest,
		"patch with malformed body":               http.StatusBadRequest,
		"patch unknown version":                   http.StatusBadRequest,
		"patch version of another application":    http.StatusBadRequest,
		"delete malformed id":                     http.StatusNotFound,
		"delete unknown":                          http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Application`)).To(Equal(2))
	g.Expect(apitest.QueryValue[bool](h,
		`SELECT name = 'app' AND versioning_strategy = 'chronological' AND NOT allow_automatic_updates
		FROM Application WHERE id = $1`, app)).To(BeTrue())
	g.Expect(apitest.QueryValue[bool](h, `SELECT archived_at IS NULL FROM ApplicationVersion WHERE id = $1`,
		otherVersion)).To(BeTrue())
}

func TestApplicationVersionErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	version := vendor.CreateApplicationVersion(app, "1.0.0")
	otherVersion := vendor.CreateApplicationVersion(vendor.CreateApplication("other"), "1.0.0")
	semverApp := apitest.Expect[apitest.Entity](vendor.Post(apitest.Applications, api.CreateApplicationRequest{
		Name: "semver", Type: types.DeploymentTypeDocker, VersioningStrategy: types.VersioningStrategySemver,
	}), http.StatusOK).ID
	kubernetesApp := apitest.Expect[apitest.Entity](vendor.Post(apitest.Applications, api.CreateApplicationRequest{
		Name: "kubernetes", Type: types.DeploymentTypeKubernetes,
	}), http.StatusOK).ID
	versionsPath := apitest.Path(apitest.Applications, app) + "/versions"
	versionPath := apitest.Path(versionsPath, version)
	g := NewWithT(t)

	create := func(name string, body any) request {
		return request{name, http.MethodPost, versionsPath, body}
	}
	compose := func(content string) map[string]string { return map[string]string{"composefile": content} }
	named := func(name string) api.CreateApplicationVersionRequest {
		return api.CreateApplicationVersionRequest{Name: name}
	}
	g.Expect(statuses(vendor.Client, []request{
		create("duplicate name", versionForm(named("1.0.0"), compose(composeFile))),
		create("no name", versionForm(named(""), compose(composeFile))),
		create("invalid compose file", versionForm(named("2"), compose("services: ["))),
		create("compose file is no mapping", versionForm(named("2"), compose("just a string"))),
		create("no compose file", versionForm(named("2"), nil)),
		create("too large compose file", versionForm(named("2"),
			compose(composeFile+"# "+strings.Repeat("x", 100*1024)+"\n"))),
		create("compose file as text/plain", apitest.Multipart{
			Fields:          map[string]any{"applicationversion": named("2")},
			Files:           compose(composeFile),
			FileContentType: "text/plain",
		}),
		create("malformed version field", apitest.Multipart{
			Fields: map[string]any{"applicationversion": "{"}, Files: compose(composeFile),
		}),
		create("no version field", apitest.Multipart{Files: compose(composeFile)}),
		create("JSON body", named("2")),
		{
			"unknown application", http.MethodPost, apitest.Path(apitest.Applications, uuid.New()) + "/versions",
			versionForm(named("2"), compose(composeFile)),
		},
		{
			"no SemVer name", http.MethodPost, apitest.Path(apitest.Applications, semverApp) + "/versions",
			versionForm(named("latest"), compose(composeFile)),
		},
		{
			"Helm chart without chart", http.MethodPost, apitest.Path(apitest.Applications, kubernetesApp) + "/versions",
			versionForm(named("1.0.0"), nil),
		},
		{
			"invalid values file", http.MethodPost, apitest.Path(apitest.Applications, kubernetesApp) + "/versions",
			versionForm(api.CreateApplicationVersionRequest{
				Name:         "1.0.0",
				ChartType:    new(types.HelmChartTypeOCI),
				ChartUrl:     new("oci://ghcr.io/distr-sh/chart"),
				ChartVersion: new("1.0.0"),
			}, map[string]string{"valuesfile": "a: ["}),
		},
		{"get malformed id", http.MethodGet, versionsPath + "/nope", nil},
		{"get unknown", http.MethodGet, apitest.Path(versionsPath, uuid.New()), nil},
		{"get version of another application", http.MethodGet, apitest.Path(versionsPath, otherVersion), nil},
		{
			"compose file of another application", http.MethodGet,
			apitest.Path(versionsPath, otherVersion) + "/compose-file", nil,
		},
		{"put without name", http.MethodPut, versionPath, api.UpdateApplicationVersionRequest{}},
		{"put with malformed body", http.MethodPut, versionPath, "{"},
		{"put malformed id", http.MethodPut, versionsPath + "/nope", api.UpdateApplicationVersionRequest{Name: "x"}},
	})).To(Equal(map[string]int{
		"duplicate name":                      http.StatusBadRequest,
		"no name":                             http.StatusBadRequest,
		"invalid compose file":                http.StatusBadRequest,
		"compose file is no mapping":          http.StatusBadRequest,
		"no compose file":                     http.StatusBadRequest,
		"too large compose file":              http.StatusRequestEntityTooLarge,
		"compose file as text/plain":          http.StatusUnsupportedMediaType,
		"malformed version field":             http.StatusBadRequest,
		"no version field":                    http.StatusBadRequest,
		"JSON body":                           http.StatusBadRequest,
		"unknown application":                 http.StatusNotFound,
		"no SemVer name":                      http.StatusBadRequest,
		"Helm chart without chart":            http.StatusBadRequest,
		"invalid values file":                 http.StatusBadRequest,
		"get malformed id":                    http.StatusNotFound,
		"get unknown":                         http.StatusNotFound,
		"get version of another application":  http.StatusNotFound,
		"compose file of another application": http.StatusNotFound,
		"put without name":                    http.StatusBadRequest,
		"put with malformed body":             http.StatusBadRequest,
		"put malformed id":                    http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM ApplicationVersion`)).To(Equal(2))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM ApplicationVersion WHERE id = $1`, version)).
		To(Equal("1.0.0"))

	vendor.Post(versionsPath, apitest.Multipart{
		Fields:          map[string]any{"applicationversion": named("2.0.0")},
		Files:           compose(composeFile),
		FileContentType: "application/json",
	}).ExpectStatus(http.StatusOK)
}

func TestUpdatingAnUnknownApplicationVersionAnswers404(t *testing.T) {
	knownBug(t, "updateApplicationVersion answers an unknown version, or one of another application, "+
		"with 400 and an empty body")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	otherVersion := vendor.CreateApplicationVersion(vendor.CreateApplication("other"), "1.0.0")
	versionsPath := apitest.Path(apitest.Applications, app) + "/versions"

	NewWithT(t).Expect(statuses(vendor.Client, []request{
		{"unknown", http.MethodPut, apitest.Path(versionsPath, uuid.New()), api.UpdateApplicationVersionRequest{Name: "x"}},
		{
			"another application's", http.MethodPut, apitest.Path(versionsPath, otherVersion),
			api.UpdateApplicationVersionRequest{Name: "x"},
		},
	})).To(HaveEach(http.StatusNotFound))
}

func TestApplicationInUseCannotBeDeleted(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	app := vendor.CreateApplication("app")
	deployed := deployDockerTarget(vendor, vendor.CreateApplicationVersion(app, "1.0.0"), "")
	g := NewWithT(t)

	vendor.Delete(apitest.Path(apitest.Applications, app)).ExpectStatus(http.StatusBadRequest)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Application WHERE id = $1`, app)).To(Equal(1))

	vendor.Delete(apitest.Path(apitest.DeploymentTargets, deployed.target)).ExpectStatus(http.StatusNoContent)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Deployment WHERE id = $1`, deployed.deployment)).
		To(BeZero())
	vendor.Delete(apitest.Path(apitest.Applications, app)).ExpectStatus(http.StatusNoContent)
	vendor.Get(apitest.Path(apitest.Applications, app)).ExpectStatus(http.StatusNotFound)
}

func TestDeploymentTargetErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	otherCustomer := h.RegisterVendor().CreateCustomerOrganization("Other Customer")
	target := vendor.CreateDeploymentTarget("target", nil)
	targetPath := apitest.Path(apitest.DeploymentTargets, target)
	unknownPath := apitest.Path(apitest.DeploymentTargets, uuid.New())
	g := NewWithT(t)

	named := api.DeploymentTargetSettings{Name: "x"}
	create := func(name string, body any) request {
		return request{name, http.MethodPost, apitest.DeploymentTargets, body}
	}
	put := func(name string, settings api.DeploymentTargetSettings) request {
		return request{
			name, http.MethodPut, targetPath,
			api.UpdateDeploymentTargetRequest{DeploymentTargetSettings: settings},
		}
	}
	g.Expect(statuses(vendor.Client, []request{
		create("no type", api.CreateDeploymentTargetRequest{DeploymentTargetSettings: named}),
		create("unknown type", api.CreateDeploymentTargetRequest{DeploymentTargetSettings: named, Type: "foo"}),
		create("Docker with resources", dockerTarget(api.DeploymentTargetSettings{
			Name: "x", Resources: &api.DeploymentTargetResources{},
		})),
		create("Docker with scope", api.CreateDeploymentTargetRequest{
			DeploymentTargetSettings: named, Type: types.DeploymentTypeDocker,
			Scope: new(types.DeploymentTargetScopeNamespace),
		}),
		create("Docker with invalid endpoint", dockerTarget(api.DeploymentTargetSettings{
			Name: "x", DockerEndpoint: new("ftp://docker"),
		})),
		create("Kubernetes with image cleanup", kubernetesTarget(api.DeploymentTargetSettings{
			Name: "x", ImageCleanupEnabled: true,
		}, new("ns"))),
		create("Kubernetes without namespace", kubernetesTarget(named, nil)),
		create("Kubernetes without scope", api.CreateDeploymentTargetRequest{
			DeploymentTargetSettings: named, Type: types.DeploymentTypeKubernetes, Namespace: new("ns"),
		}),
		create("Kubernetes with invalid CPU limit", kubernetesTarget(api.DeploymentTargetSettings{
			Name: "x", Resources: &api.DeploymentTargetResources{CPULimit: "lots"},
		}, new("ns"))),
		create("unknown customer", api.CreateDeploymentTargetRequest{
			DeploymentTargetSettings: named, Type: types.DeploymentTypeDocker, CustomerOrganizationID: new(uuid.New()),
		}),
		create("another vendor's customer", api.CreateDeploymentTargetRequest{
			DeploymentTargetSettings: named, Type: types.DeploymentTypeDocker, CustomerOrganizationID: &otherCustomer,
		}),
		create("malformed body", "{"),
		{"get malformed id", http.MethodGet, apitest.DeploymentTargets + "/nope", nil},
		{"get unknown", http.MethodGet, unknownPath, nil},
		put("put Docker with resources", api.DeploymentTargetSettings{
			Name: "target", Resources: &api.DeploymentTargetResources{},
		}),
		{"put malformed body", http.MethodPut, targetPath, "{"},
		{"put unknown", http.MethodPut, unknownPath, api.UpdateDeploymentTargetRequest{DeploymentTargetSettings: named}},
		{"delete malformed id", http.MethodDelete, apitest.DeploymentTargets + "/nope", nil},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
		{"access request for unknown", http.MethodPost, unknownPath + "/access-request", nil},
	})).To(Equal(map[string]int{
		"no type":                           http.StatusBadRequest,
		"unknown type":                      http.StatusBadRequest,
		"Docker with resources":             http.StatusBadRequest,
		"Docker with scope":                 http.StatusBadRequest,
		"Docker with invalid endpoint":      http.StatusBadRequest,
		"Kubernetes with image cleanup":     http.StatusBadRequest,
		"Kubernetes without namespace":      http.StatusBadRequest,
		"Kubernetes without scope":          http.StatusBadRequest,
		"Kubernetes with invalid CPU limit": http.StatusBadRequest,
		"unknown customer":                  http.StatusForbidden,
		"another vendor's customer":         http.StatusForbidden,
		"malformed body":                    http.StatusBadRequest,
		"get malformed id":                  http.StatusNotFound,
		"get unknown":                       http.StatusNotFound,
		"put Docker with resources":         http.StatusBadRequest,
		"put malformed body":                http.StatusBadRequest,
		"put unknown":                       http.StatusNotFound,
		"delete malformed id":               http.StatusNotFound,
		"delete unknown":                    http.StatusNotFound,
		"access request for unknown":        http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM DeploymentTarget`)).To(Equal(1))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM DeploymentTarget WHERE id = $1`, target)).
		To(Equal("target"))
}

func TestUpdatingADeploymentTargetWithAnUnknownControllerVersionAnswers400(t *testing.T) {
	knownBug(t, "answers 500 because db.UpdateDeploymentTarget does not map the foreign key violation "+
		"of an unknown controller version")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	target := vendor.CreateDeploymentTarget("target", nil)

	vendor.Put(apitest.Path(apitest.DeploymentTargets, target), api.UpdateDeploymentTargetRequest{
		DeploymentTargetSettings: api.DeploymentTargetSettings{Name: "target"},
		ControllerVersionID:      new(uuid.New()),
	}).ExpectStatus(http.StatusBadRequest)
}

func TestSecretErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	customer := vendor.CreateCustomerOrganization("Customer")
	secret := vendor.CreateSecret("EXISTING", "value", nil)
	unknownPath := apitest.Path(apitest.Secrets, uuid.New())
	g := NewWithT(t)

	create := func(name, key, value string, customerOrganizationID *uuid.UUID) request {
		return request{name, http.MethodPost, apitest.Secrets, api.CreateSecretRequest{
			Key: key, Value: value, CustomerOrganizationID: customerOrganizationID,
		}}
	}
	g.Expect(statuses(vendor.Client, []request{
		create("no key", "", "value", nil),
		create("key with space", "HAS SPACE", "value", nil),
		create("key with dash", "HAS-DASH", "value", nil),
		create("key starting with digit", "1KEY", "value", nil),
		create("no value", "EMPTY", "", nil),
		create("duplicate key", "EXISTING", "value", nil),
		create("unknown customer", "KEY", "value", new(uuid.New())),
		{"create malformed body", http.MethodPost, apitest.Secrets, "{"},
		{"put malformed id", http.MethodPut, apitest.Secrets + "/nope", api.UpdateSecretRequest{Value: "v"}},
		{"put unknown", http.MethodPut, unknownPath, api.UpdateSecretRequest{Value: "v"}},
		{"put malformed body", http.MethodPut, apitest.Path(apitest.Secrets, secret), "{"},
		{"delete malformed id", http.MethodDelete, apitest.Secrets + "/nope", nil},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"no key":                  http.StatusBadRequest,
		"key with space":          http.StatusBadRequest,
		"key with dash":           http.StatusBadRequest,
		"key starting with digit": http.StatusBadRequest,
		"no value":                http.StatusBadRequest,
		"duplicate key":           http.StatusConflict,
		"unknown customer":        http.StatusBadRequest,
		"create malformed body":   http.StatusBadRequest,
		"put malformed id":        http.StatusBadRequest,
		"put unknown":             http.StatusNotFound,
		"put malformed body":      http.StatusBadRequest,
		"delete malformed id":     http.StatusBadRequest,
		"delete unknown":          http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Secret`)).To(Equal(1))

	vendor.CreateSecret("EXISTING", "customer value", &customer)
	vendor.Delete(apitest.Path(apitest.Secrets, secret)).ExpectStatus(http.StatusNoContent)
	vendor.Delete(apitest.Path(apitest.Secrets, secret)).ExpectStatus(http.StatusNotFound)
}

func TestChangingASecretInUseNeedsConfirmation(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	secret := vendor.CreateSecret("TOKEN", "old", nil)
	deployed := deployDockerTarget(vendor,
		vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0"), "TOKEN={{ .Secrets.TOKEN }}\n")
	secretPath := apitest.Path(apitest.Secrets, secret)
	g := NewWithT(t)

	envFile := func() string {
		t.Helper()
		resources := apitest.Expect[api.ControllerResource](
			h.LoginController(vendor.RequestAccess(deployed.target)).Get(controllerResources), http.StatusOK)
		g.Expect(resources.Deployments).To(HaveLen(1))
		return string(resources.Deployments[0].EnvFile)
	}
	expectConflict := func(r *apitest.Response) {
		t.Helper()
		conflict := apitest.Expect[api.AffectedDeploymentsConflictResponse](r, http.StatusConflict)
		g.Expect(conflict.AffectedDeployments).To(HaveLen(1))
		g.Expect(conflict.AffectedDeployments[0].DeploymentID).To(Equal(deployed.deployment))
		g.Expect(conflict.AffectedDeployments[0].DeploymentTargetID).To(Equal(deployed.target))
	}

	expectConflict(vendor.Put(secretPath, api.UpdateSecretRequest{Value: "new"}))
	expectConflict(vendor.Delete(secretPath))
	g.Expect(envFile()).To(Equal("TOKEN=old\n"))

	vendor.Put(secretPath+"?confirm=true", api.UpdateSecretRequest{Value: "new"}).ExpectStatus(http.StatusOK)
	g.Expect(envFile()).To(Equal("TOKEN=new\n"))
}

func TestCustomerOrganizationErrors(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	customer := vendor.CreateCustomerOrganization("Customer")
	customerPath := apitest.Path(apitest.CustomerOrganizations, customer)
	unknownPath := apitest.Path(apitest.CustomerOrganizations, uuid.New())
	g := NewWithT(t)

	alerts := []types.CustomerOrganizationFeature{types.CustomerOrganizationFeatureAlerts}
	g.Expect(statuses(vendor.Client, []request{
		{
			"create alerts without deployment targets", http.MethodPost, apitest.CustomerOrganizations,
			api.CreateUpdateCustomerOrganizationRequest{Name: "x", Features: alerts},
		},
		{
			"create unknown feature", http.MethodPost, apitest.CustomerOrganizations,
			api.CreateUpdateCustomerOrganizationRequest{Name: "x", Features: []types.CustomerOrganizationFeature{"bogus"}},
		},
		{
			"create unknown image", http.MethodPost, apitest.CustomerOrganizations,
			api.CreateUpdateCustomerOrganizationRequest{Name: "x", ImageID: new(uuid.New())},
		},
		{"create malformed body", http.MethodPost, apitest.CustomerOrganizations, "{"},
		{
			"put alerts without deployment targets", http.MethodPut, customerPath,
			api.CreateUpdateCustomerOrganizationRequest{Name: "x", Features: alerts},
		},
		{"put malformed body", http.MethodPut, customerPath, "{"},
		{
			"put malformed id", http.MethodPut, apitest.CustomerOrganizations + "/nope",
			api.CreateUpdateCustomerOrganizationRequest{Name: "x"},
		},
		{"delete malformed id", http.MethodDelete, apitest.CustomerOrganizations + "/nope", nil},
		{"delete unknown", http.MethodDelete, unknownPath, nil},
	})).To(Equal(map[string]int{
		"create alerts without deployment targets": http.StatusBadRequest,
		"create unknown feature":                   http.StatusBadRequest,
		"create unknown image":                     http.StatusBadRequest,
		"create malformed body":                    http.StatusBadRequest,
		"put alerts without deployment targets":    http.StatusBadRequest,
		"put malformed body":                       http.StatusBadRequest,
		"put malformed id":                         http.StatusNotFound,
		"delete malformed id":                      http.StatusNotFound,
		"delete unknown":                           http.StatusNotFound,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM CustomerOrganization`)).To(Equal(1))
	g.Expect(apitest.QueryValue[string](h, `SELECT name FROM CustomerOrganization WHERE id = $1`, customer)).
		To(Equal("Customer"))
}

func TestCreatingACustomerOrganizationAppliesItsFeatures(t *testing.T) {
	knownBug(t, "POST /customer-organizations validates the features of the request and then ignores them, "+
		"so a customer created with alerts through the API or the SDK ends up with the defaults")
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	g := NewWithT(t)

	created := apitest.Expect[api.CustomerOrganization](vendor.Post(apitest.CustomerOrganizations,
		api.CreateUpdateCustomerOrganizationRequest{Name: "x", Features: []types.CustomerOrganizationFeature{
			types.CustomerOrganizationFeatureDeploymentTargets, types.CustomerOrganizationFeatureAlerts,
		}}), http.StatusOK)
	g.Expect(created.Features).To(ConsistOf(
		types.CustomerOrganizationFeatureDeploymentTargets, types.CustomerOrganizationFeatureAlerts))
}

func TestDeletingACustomerOrganizationRemovesWhatBelongsToIt(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	customer := vendor.CreateCustomerOrganization("Customer")
	target := vendor.CreateDeploymentTarget("customer target", &customer)
	secret := vendor.CreateSecret("KEY", "value", &customer)
	customerAdmin := vendor.Invite(types.UserRoleAdmin, &customer)
	vendorTarget := vendor.CreateDeploymentTarget("vendor target", nil)
	g := NewWithT(t)

	vendor.Delete(apitest.Path(apitest.CustomerOrganizations, customer)).ExpectStatus(http.StatusNoContent)

	customerAdmin.Get(contextPath).ExpectStatus(http.StatusUnauthorized)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM DeploymentTarget WHERE id = $1`, target)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Secret WHERE id = $1`, secret)).To(BeZero())
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Organization_UserAccount WHERE user_account_id = $1`,
		customerAdmin.ID)).To(BeZero())
	g.Expect(apitest.IDs(vendor.Get(apitest.DeploymentTargets), http.StatusOK)).To(Equal([]uuid.UUID{vendorTarget}))
	vendor.Delete(apitest.Path(apitest.CustomerOrganizations, customer)).ExpectStatus(http.StatusNotFound)
}
