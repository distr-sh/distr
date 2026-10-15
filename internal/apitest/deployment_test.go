package apitest_test

import (
	"net/http"
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apitest"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func deploy(name string, body api.DeploymentRequest) request {
	return request{name, http.MethodPut, apitest.Deployments, body}
}

func revisions(deployment uuid.UUID) string {
	return apitest.Path(apitest.Deployments, deployment) + "/revisions"
}

func createEntitlement(vendor *apitest.User, application, customerOrganization uuid.UUID) uuid.UUID {
	return apitest.Expect[apitest.Entity](vendor.Post(apitest.ApplicationEntitlements,
		types.ApplicationEntitlementWithVersions{ApplicationEntitlementBase: types.ApplicationEntitlementBase{
			Name:                   "entitlement",
			ApplicationID:          application,
			CustomerOrganizationID: &customerOrganization,
		}},
	), http.StatusOK).ID
}

func TestDeploymentRequestValidation(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	target := vendor.CreateDeploymentTarget("docker", nil)
	kubernetesTarget := apitest.Expect[apitest.Entity](vendor.Post(apitest.DeploymentTargets,
		api.CreateDeploymentTargetRequest{
			DeploymentTargetSettings: api.DeploymentTargetSettings{Name: "kubernetes"},
			Type:                     types.DeploymentTypeKubernetes,
			Namespace:                new("distr"),
			Scope:                    new(types.DeploymentTargetScopeNamespace),
		}), http.StatusOK).ID
	otherVendor := h.RegisterVendor()
	foreignVersion := otherVendor.CreateApplicationVersion(otherVendor.CreateApplication("foreign"), "1.0.0")
	foreignTarget := otherVendor.CreateDeploymentTarget("foreign", nil)
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		deploy("unknown version", api.DeploymentRequest{DeploymentTargetID: target, ApplicationVersionID: uuid.New()}),
		deploy("unknown target", api.DeploymentRequest{DeploymentTargetID: uuid.New(), ApplicationVersionID: version}),
		deploy("other vendor's version",
			api.DeploymentRequest{DeploymentTargetID: target, ApplicationVersionID: foreignVersion}),
		deploy("other vendor's target",
			api.DeploymentRequest{DeploymentTargetID: foreignTarget, ApplicationVersionID: version}),
		deploy("docker version to kubernetes target",
			api.DeploymentRequest{DeploymentTargetID: kubernetesTarget, ApplicationVersionID: version}),
		deploy("unknown deployment", api.DeploymentRequest{
			DeploymentID: new(uuid.New()), DeploymentTargetID: target, ApplicationVersionID: version,
		}),
		deploy("unknown secret in env file", api.DeploymentRequest{
			DeploymentTargetID: target, ApplicationVersionID: version,
			EnvFileData: []byte("API_KEY={{ .Secrets.MISSING }}\n"),
		}),
		deploy("invalid env file template", api.DeploymentRequest{
			DeploymentTargetID: target, ApplicationVersionID: version, EnvFileData: []byte("API_KEY={{ .Secrets.\n"),
		}),
		deploy("ignore revision skew on docker", api.DeploymentRequest{
			DeploymentTargetID: target, ApplicationVersionID: version, IgnoreRevisionSkew: true,
		}),
		deploy("unknown entitlement", api.DeploymentRequest{
			DeploymentTargetID: target, ApplicationVersionID: version, ApplicationEntitlementID: new(uuid.New()),
		}),
		deploy("automatic updates without the feature", api.DeploymentRequest{
			DeploymentTargetID: target, ApplicationVersionID: version, AutomaticApplicationUpdatesEnabled: new(true),
		}),
		{"malformed body", http.MethodPut, apitest.Deployments, "not an object"},
		{"no body", http.MethodPut, apitest.Deployments, nil},
	})).To(Equal(map[string]int{
		"unknown version":                       http.StatusBadRequest,
		"unknown target":                        http.StatusBadRequest,
		"other vendor's version":                http.StatusBadRequest,
		"other vendor's target":                 http.StatusBadRequest,
		"docker version to kubernetes target":   http.StatusBadRequest,
		"unknown deployment":                    http.StatusBadRequest,
		"unknown secret in env file":            http.StatusBadRequest,
		"invalid env file template":             http.StatusBadRequest,
		"ignore revision skew on docker":        http.StatusBadRequest,
		"unknown entitlement":                   http.StatusBadRequest,
		"automatic updates without the feature": http.StatusBadRequest,
		"malformed body":                        http.StatusBadRequest,
		"no body":                               http.StatusBadRequest,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Deployment`)).To(Equal(0))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM DeploymentRevision`)).To(Equal(0))
}

func TestDeploymentLifecycle(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	application := vendor.CreateApplication("app")
	first := vendor.CreateApplicationVersion(application, "1.0.0")
	second := vendor.CreateApplicationVersion(application, "2.0.0")
	otherApplication := vendor.CreateApplicationVersion(vendor.CreateApplication("other"), "1.0.0")
	target := vendor.CreateDeploymentTarget("target", nil)
	g := NewWithT(t)

	deployment, firstRevision := vendor.Deploy(api.DeploymentRequest{
		DeploymentTargetID: target, ApplicationVersionID: first, DockerType: new(types.DockerTypeCompose),
	})
	g.Expect(apitest.QueryValue[uuid.UUID](h,
		`SELECT created_by_user_account_id FROM DeploymentRevision WHERE id = $1`, firstRevision)).To(Equal(vendor.ID))
	_, secondRevision := vendor.Deploy(api.DeploymentRequest{
		DeploymentID: &deployment, DeploymentTargetID: target, ApplicationVersionID: second,
	})
	g.Expect(secondRevision).NotTo(Equal(firstRevision))
	g.Expect(apitest.QueryValue[uuid.UUID](h,
		`SELECT application_version_id FROM DeploymentRevision WHERE id = $1`, secondRevision)).To(Equal(second))
	g.Expect(apitest.IDs(vendor.Get(revisions(deployment)), http.StatusOK)).
		To(ConsistOf(firstRevision, secondRevision))

	vendor.Put(apitest.Deployments, api.DeploymentRequest{
		DeploymentID: &deployment, DeploymentTargetID: target, ApplicationVersionID: otherApplication,
	}).ExpectStatus(http.StatusBadRequest)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM DeploymentRevision`)).To(Equal(2))

	vendor.Deploy(api.DeploymentRequest{DeploymentTargetID: target, ApplicationVersionID: otherApplication})
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Deployment WHERE deployment_target_id = $1`, target)).
		To(Equal(2), "a target holds several deployments")

	vendor.Delete(apitest.Path(apitest.Deployments, deployment)).ExpectStatus(http.StatusOK)
	vendor.Delete(apitest.Path(apitest.Deployments, deployment)).ExpectStatus(http.StatusNotFound)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Deployment WHERE id = $1`, deployment)).To(Equal(0))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM DeploymentRevision WHERE deployment_id = $1`,
		deployment)).To(Equal(0))
}

func TestDeploymentAccess(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	target := vendor.CreateDeploymentTarget("target", nil)
	deployment, _ := vendor.Deploy(api.DeploymentRequest{DeploymentTargetID: target, ApplicationVersionID: version})
	redeploy := api.DeploymentRequest{DeploymentID: &deployment, DeploymentTargetID: target, ApplicationVersionID: version}
	reader := vendor.Invite(types.UserRoleReadOnly, nil)
	otherVendor := h.RegisterVendor()
	g := NewWithT(t)

	g.Expect(statuses(otherVendor.Client, []request{
		{"revisions", http.MethodGet, revisions(deployment), nil},
		{"metrics", http.MethodGet, apitest.Path(apitest.Deployments, deployment) + "/metrics", nil},
		{"logs", http.MethodGet, apitest.Path(apitest.Deployments, deployment) + "/logs?resource=app", nil},
		{"delete", http.MethodDelete, apitest.Path(apitest.Deployments, deployment), nil},
		deploy("redeploy", redeploy),
	})).To(Equal(map[string]int{
		"revisions": http.StatusNotFound,
		"metrics":   http.StatusNotFound,
		"logs":      http.StatusNotFound,
		"delete":    http.StatusNotFound,
		"redeploy":  http.StatusBadRequest,
	}))
	g.Expect(statuses(reader.Client, []request{
		{"revisions", http.MethodGet, revisions(deployment), nil},
		{"delete", http.MethodDelete, apitest.Path(apitest.Deployments, deployment), nil},
		deploy("redeploy", redeploy),
		deploy("new deployment", api.DeploymentRequest{DeploymentTargetID: target, ApplicationVersionID: version}),
	})).To(Equal(map[string]int{
		"revisions":      http.StatusOK,
		"delete":         http.StatusForbidden,
		"redeploy":       http.StatusForbidden,
		"new deployment": http.StatusForbidden,
	}))
	g.Expect(statuses(vendor.Client, []request{
		{"unknown deployment", http.MethodGet, revisions(uuid.New()), nil},
		{"malformed deployment", http.MethodGet, "/api/v1/deployments/not-a-uuid/revisions", nil},
		{
			"patch automatic updates without the feature", http.MethodPatch, apitest.Path(apitest.Deployments, deployment),
			api.PatchDeploymentRequest{AutomaticApplicationUpdatesEnabled: new(true)},
		},
	})).To(Equal(map[string]int{
		"unknown deployment":                          http.StatusNotFound,
		"malformed deployment":                        http.StatusNotFound,
		"patch automatic updates without the feature": http.StatusForbidden,
	}))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM DeploymentRevision`)).To(Equal(1))
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Deployment WHERE id = $1`, deployment)).To(Equal(1))
}

func TestCustomerDeployments(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	entitled := vendor.CreateApplication("entitled")
	entitledVersion := vendor.CreateApplicationVersion(entitled, "1.0.0")
	unentitledVersion := vendor.CreateApplicationVersion(vendor.CreateApplication("unentitled"), "1.0.0")
	vendorTarget := vendor.CreateDeploymentTarget("vendor", nil)
	own := vendor.CreateCustomerOrganization("own")
	other := vendor.CreateCustomerOrganization("other")
	customer := vendor.Invite(types.UserRoleAdmin, &own)
	ownTarget := customer.CreateDeploymentTarget("own", nil)
	otherTarget := vendor.CreateDeploymentTarget("other", &other)
	g := NewWithT(t)

	g.Expect(statuses(customer.Client, []request{
		deploy("to vendor's target",
			api.DeploymentRequest{DeploymentTargetID: vendorTarget, ApplicationVersionID: entitledVersion}),
		deploy("to other customer's target",
			api.DeploymentRequest{DeploymentTargetID: otherTarget, ApplicationVersionID: entitledVersion}),
	})).To(Equal(map[string]int{
		"to vendor's target":         http.StatusBadRequest,
		"to other customer's target": http.StatusBadRequest,
	}))
	customer.Deploy(api.DeploymentRequest{DeploymentTargetID: ownTarget, ApplicationVersionID: unentitledVersion})

	ownEntitlement := createEntitlement(vendor, entitled, own)
	otherEntitlement := createEntitlement(vendor, entitled, other)
	g.Expect(statuses(customer.Client, []request{
		deploy("without entitlement once entitlements exist",
			api.DeploymentRequest{DeploymentTargetID: ownTarget, ApplicationVersionID: entitledVersion}),
		deploy("with other customer's entitlement", api.DeploymentRequest{
			DeploymentTargetID: ownTarget, ApplicationVersionID: entitledVersion,
			ApplicationEntitlementID: &otherEntitlement,
		}),
		deploy("entitlement for another application", api.DeploymentRequest{
			DeploymentTargetID: ownTarget, ApplicationVersionID: unentitledVersion,
			ApplicationEntitlementID: &ownEntitlement,
		}),
	})).To(Equal(map[string]int{
		"without entitlement once entitlements exist": http.StatusBadRequest,
		"with other customer's entitlement":           http.StatusBadRequest,
		"entitlement for another application":         http.StatusBadRequest,
	}))
	g.Expect(statuses(vendor.Client, []request{
		deploy("entitlement to vendor's target", api.DeploymentRequest{
			DeploymentTargetID: vendorTarget, ApplicationVersionID: entitledVersion,
			ApplicationEntitlementID: &ownEntitlement,
		}),
		deploy("entitlement to target of another customer", api.DeploymentRequest{
			DeploymentTargetID: otherTarget, ApplicationVersionID: entitledVersion,
			ApplicationEntitlementID: &ownEntitlement,
		}),
	})).To(Equal(map[string]int{
		"entitlement to vendor's target":            http.StatusBadRequest,
		"entitlement to target of another customer": http.StatusBadRequest,
	}))

	deployment, revision := customer.Deploy(api.DeploymentRequest{
		DeploymentTargetID: ownTarget, ApplicationVersionID: entitledVersion, ApplicationEntitlementID: &ownEntitlement,
	})
	g.Expect(apitest.QueryValue[*uuid.UUID](h,
		`SELECT application_entitlement_id FROM Deployment WHERE id = $1`, deployment)).
		To(HaveValue(Equal(ownEntitlement)))
	g.Expect(apitest.QueryValue[uuid.UUID](h,
		`SELECT created_by_user_account_id FROM DeploymentRevision WHERE id = $1`, revision)).To(Equal(customer.ID))
	vendor.Deploy(api.DeploymentRequest{DeploymentTargetID: otherTarget, ApplicationVersionID: unentitledVersion})

	otherCustomer := vendor.Invite(types.UserRoleAdmin, &other)
	g.Expect(statuses(otherCustomer.Client, []request{
		{"revisions", http.MethodGet, revisions(deployment), nil},
		{"delete", http.MethodDelete, apitest.Path(apitest.Deployments, deployment), nil},
	})).To(Equal(map[string]int{
		"revisions": http.StatusNotFound,
		"delete":    http.StatusNotFound,
	}))
	vendor.Get(revisions(deployment)).ExpectStatus(http.StatusOK)
	g.Expect(apitest.QueryValue[int](h, `SELECT count(*) FROM Deployment WHERE id = $1`, deployment)).To(Equal(1))
}

func TestDeploymentSecretsAreScopedToTheTarget(t *testing.T) {
	t.Parallel()
	h := apitest.New(t)
	vendor := h.RegisterVendor()
	version := vendor.CreateApplicationVersion(vendor.CreateApplication("app"), "1.0.0")
	own := vendor.CreateCustomerOrganization("own")
	other := vendor.CreateCustomerOrganization("other")
	vendor.CreateSecret("VENDOR", "vendor-value", nil)
	vendor.CreateSecret("OWN", "own-value", &own)
	vendor.CreateSecret("OTHER", "other-value", &other)
	ownTarget := vendor.CreateDeploymentTarget("own", &own)
	vendorTarget := vendor.CreateDeploymentTarget("vendor", nil)
	withEnv := func(target uuid.UUID, envFile string) api.DeploymentRequest {
		return api.DeploymentRequest{
			DeploymentTargetID: target, ApplicationVersionID: version, EnvFileData: []byte(envFile),
		}
	}
	g := NewWithT(t)

	g.Expect(statuses(vendor.Client, []request{
		deploy("other customer's secret on customer target", withEnv(ownTarget, "A={{ .Secrets.OTHER }}\n")),
		deploy("customer secret on vendor target", withEnv(vendorTarget, "A={{ .Secrets.OWN }}\n")),
		deploy("vendor secret on customer target", withEnv(ownTarget, "A={{ .Secrets.VENDOR }}\n")),
	})).To(Equal(map[string]int{
		"other customer's secret on customer target": http.StatusBadRequest,
		"customer secret on vendor target":           http.StatusBadRequest,
		"vendor secret on customer target":           http.StatusBadRequest,
	}))

	vendor.Deploy(withEnv(ownTarget, "A={{ .Secrets.OWN }}\n"))
	vendor.Deploy(withEnv(vendorTarget, "A={{ .Secrets.VENDOR }}\n"))
	for target, envFile := range map[uuid.UUID]string{ownTarget: "A=own-value\n", vendorTarget: "A=vendor-value\n"} {
		resources := apitest.Expect[api.ControllerResource](
			h.LoginController(vendor.RequestAccess(target)).Get(controllerResources), http.StatusOK)
		g.Expect(resources.Deployments).To(HaveLen(1))
		g.Expect(string(resources.Deployments[0].EnvFile)).To(Equal(envFile))
	}
}
