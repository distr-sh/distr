package handlers

import (
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
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
		middleware.RequireVendor,
		middleware.LicensingFeatureFlagEnabledMiddleware,
	)
	r.Get("/", getLicenses).
		With(option.Description("List all licenses grouped by customer")).
		With(option.Response(http.StatusOK, []licenseResponse{}))
}

func getLicenses(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	authInfo := auth.Authentication.Require(ctx)
	orgID := *authInfo.CurrentOrgID()

	customers, err := db.GetCustomerOrganizationsByOrganizationID(ctx, orgID)
	if err != nil {
		log.Error("failed to get customer organizations", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(
			w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	appEntitlements, err := db.GetApplicationEntitlementsWithOrganizationID(
		ctx, orgID, nil,
	)
	if err != nil {
		log.Error("failed to get application entitlements", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(
			w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	artifactEntitlements, err := db.GetArtifactEntitlements(ctx, orgID)
	if err != nil {
		log.Error("failed to get artifact entitlements", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(
			w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

	licenseKeys, err := db.GetLicenseKeys(ctx, orgID)
	if err != nil {
		log.Error("failed to get license keys", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(
			w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError,
		)
		return
	}

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

	RespondJSON(w, licenses)
}
