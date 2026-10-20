package handlers

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

type licenseResponse struct {
	CustomerOrganization    api.CustomerOrganization       `json:"customerOrganization"`
	ApplicationEntitlements []types.ApplicationEntitlement `json:"applicationEntitlements"`
	ArtifactEntitlements    []types.ArtifactEntitlement    `json:"artifactEntitlements"`
	LicenseKeys             []types.LicenseKey             `json:"licenseKeys"`
}

func LicensesRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Licensing"))
	r.Use(
		middleware.RequireOrgAndRole,
		middleware.RequireVendorOrPartner,
		middleware.LicensingFeatureFlagEnabledMiddleware,
	)
	r.Get("/", respond.Handler(getLicenses)).
		With(option.Description("List all licenses grouped by customer")).
		With(option.Response(http.StatusOK, []licenseResponse{}))
}

func getLicenses(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	authInfo := auth.Authentication.Require(ctx)
	orgID := *authInfo.CurrentOrgID()

	var customers []types.CustomerOrganizationWithUsage
	var err error
	if partnerOrgID := authInfo.CurrentPartnerOrgID(); partnerOrgID != nil {
		customers, err = db.GetCustomerOrganizationsByPartnerOrgID(ctx, *partnerOrgID)
	} else {
		customers, err = db.GetCustomerOrganizationsByOrganizationID(ctx, orgID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get customer organizations: %w", err)
	} else if appEntitlements, err := db.GetApplicationEntitlementsWithOrganizationID(
		ctx, orgID, nil,
	); err != nil {
		return nil, fmt.Errorf("failed to get application entitlements: %w", err)
	} else if artifactEntitlements, err := db.GetArtifactEntitlements(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to get artifact entitlements: %w", err)
	} else if licenseKeys, err := db.GetLicenseKeys(ctx, orgID); err != nil {
		return nil, fmt.Errorf("failed to get license keys: %w", err)
	} else {
		licenses := make([]licenseResponse, 0, len(customers))
		for _, customer := range customers {
			license := licenseResponse{
				CustomerOrganization: mapping.CustomerOrganizationToAPI(
					customer.CustomerOrganization,
				),
				ApplicationEntitlements: []types.ApplicationEntitlement{},
				ArtifactEntitlements:    []types.ArtifactEntitlement{},
				LicenseKeys:             []types.LicenseKey{},
			}
			for _, ae := range appEntitlements {
				if ae.CustomerOrganizationID != nil &&
					*ae.CustomerOrganizationID == customer.ID {
					license.ApplicationEntitlements = append(
						license.ApplicationEntitlements, ae,
					)
				}
			}
			for _, ae := range artifactEntitlements {
				if ae.CustomerOrganizationID != nil &&
					*ae.CustomerOrganizationID == customer.ID {
					license.ArtifactEntitlements = append(
						license.ArtifactEntitlements, ae,
					)
				}
			}
			for _, lk := range licenseKeys {
				if lk.CustomerOrganizationID != nil &&
					*lk.CustomerOrganizationID == customer.ID {
					license.LicenseKeys = append(license.LicenseKeys, lk)
				}
			}
			licenses = append(licenses, license)
		}

		return respond.JSON(licenses), nil
	}
}
