package apitest

import (
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

const (
	AccessTokens            = "/api/v1/settings/tokens"
	Advisories              = "/api/v1/advisories"
	AlertConfigurations     = "/api/v1/alert-configurations"
	ApplicationEntitlements = "/api/v1/application-entitlements"
	ArtifactEntitlements    = "/api/v1/artifact-entitlements"
	ArtifactPulls           = "/api/v1/artifact-pulls"
	Artifacts               = "/api/v1/artifacts"
	Applications            = "/api/v1/applications"
	CustomerOrganizations   = "/api/v1/customer-organizations"
	DeploymentTargets       = "/api/v1/deployment-targets"
	LicenseKeys             = "/api/v1/license-keys"
	LicenseTemplates        = "/api/v1/license-templates"
	NotificationRecords     = "/api/v1/notification-records"
	OrganizationWebhook     = "/api/v1/organization/webhook"
	Secrets                 = "/api/v1/secrets"
	SupportBundles          = "/api/v1/support-bundles"
	SupportBundleCollect    = "/api/v1/support-bundle-collect"
	UpdateNotifications     = "/api/v1/update-notification-configurations"
	UserAccounts            = "/api/v1/user-accounts"
)

// Path is the path of the entity id in the collection.
func Path(collection string, id uuid.UUID) string {
	return collection + "/" + id.String()
}

// Entity decodes the one field every resource has, so that a fixture does not depend on the
// shape of the response that created it.
type Entity struct {
	ID uuid.UUID `json:"id"`
}

func (u *User) CreateApplication(name string) uuid.UUID {
	u.h.t.Helper()
	return Expect[Entity](u.Post(Applications, api.CreateApplicationRequest{
		Name: name,
		Type: types.DeploymentTypeDocker,
	}), http.StatusOK).ID
}

// DockerApplicationVersion is the form creating a version of a Docker application.
func DockerApplicationVersion(name string) Multipart {
	return Multipart{
		Fields: map[string]any{"applicationversion": api.CreateApplicationVersionRequest{Name: name}},
		Files:  map[string]string{"composefile": "services:\n  app:\n    image: nginx:1.29\n"},
	}
}

// CreateApplicationVersion creates a version of the Docker application applicationID.
func (u *User) CreateApplicationVersion(applicationID uuid.UUID, name string) uuid.UUID {
	u.h.t.Helper()
	return Expect[Entity](
		u.Post(Path(Applications, applicationID)+"/versions", DockerApplicationVersion(name)),
		http.StatusOK,
	).ID
}

// CreateCustomerOrganization creates a customer organization with the features a new one gets by
// default, plus additional, which are set the way the UI does it: with an update after the creation.
func (u *User) CreateCustomerOrganization(name string, additional ...types.CustomerOrganizationFeature) uuid.UUID {
	u.h.t.Helper()
	id := Expect[Entity](u.Post(CustomerOrganizations, api.CreateUpdateCustomerOrganizationRequest{Name: name}),
		http.StatusOK).ID
	if len(additional) > 0 {
		u.Put(Path(CustomerOrganizations, id), api.CreateUpdateCustomerOrganizationRequest{
			Name: name,
			Features: append([]types.CustomerOrganizationFeature{
				types.CustomerOrganizationFeatureDeploymentTargets,
				types.CustomerOrganizationFeatureArtifacts,
			}, additional...),
		}).ExpectStatus(http.StatusOK)
	}
	return id
}

// CreateDeploymentTarget creates a Docker deployment target for customerOrganizationID, or for
// the organization of u itself if it is nil.
func (u *User) CreateDeploymentTarget(name string, customerOrganizationID *uuid.UUID) uuid.UUID {
	u.h.t.Helper()
	return Expect[Entity](u.Post(DeploymentTargets, api.CreateDeploymentTargetRequest{
		DeploymentTargetSettings: api.DeploymentTargetSettings{Name: name},
		Type:                     types.DeploymentTypeDocker,
		CustomerOrganizationID:   customerOrganizationID,
	}), http.StatusOK).ID
}

// CreateSecret creates a secret for customerOrganizationID, or for the organization of u itself
// if it is nil.
func (u *User) CreateSecret(key, value string, customerOrganizationID *uuid.UUID) uuid.UUID {
	u.h.t.Helper()
	return Expect[Entity](u.Post(Secrets, api.CreateSecretRequest{
		Key:                    key,
		Value:                  value,
		CustomerOrganizationID: customerOrganizationID,
	}), http.StatusCreated).ID
}

// IDs decodes a list response into the IDs of its entries.
func IDs(r *Response, status int) []uuid.UUID {
	r.t.Helper()
	entities := Expect[[]Entity](r, status)
	ids := make([]uuid.UUID, len(entities))
	for i, e := range entities {
		ids[i] = e.ID
	}
	return ids
}
