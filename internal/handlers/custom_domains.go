package handlers

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func CustomDomainsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Custom Domains"))
	r.Use(middleware.RequireVendor, middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", getCustomDomainsHandler).
		With(option.Description("List all custom domains of the current organization")).
		With(option.Response(http.StatusOK, []types.CustomDomain{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Post("/", createCustomDomainHandler).
			With(option.Description("Register a new custom domain for the current organization")).
			With(option.Request(api.CreateCustomDomainRequest{})).
			With(option.Response(http.StatusOK, types.CustomDomain{}))
		r.Delete("/{customDomainId}", deleteCustomDomainHandler).
			With(option.Description("Delete a custom domain")).
			With(option.Request(struct {
				CustomDomainID uuid.UUID `path:"customDomainId"`
			}{}))
	})
}

func getCustomDomainsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	customDomains, err := db.GetCustomDomains(ctx, *auth.CurrentOrgID())
	if err != nil {
		log.Error("failed to get custom domains", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	RespondJSON(w, customDomains)
}

func createCustomDomainHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	request, err := JsonBody[api.CreateCustomDomainRequest](w, r)
	if err != nil {
		return
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isPlatformOwnedDomain(request.Domain) {
		http.Error(w, "this domain is owned by the platform and can not be registered", http.StatusBadRequest)
		return
	}

	customDomain := types.CustomDomain{
		Domain:         request.Domain,
		Type:           request.DomainType,
		OrganizationID: *auth.CurrentOrgID(),
	}
	if err := db.CreateCustomDomain(ctx, &customDomain); errors.Is(err, apierrors.ErrConflict) {
		http.Error(w, "this domain is already in use", http.StatusConflict)
	} else if err != nil {
		log.Error("failed to create custom domain", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, customDomain)
	}
}

func deleteCustomDomainHandler(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("customDomainId"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	if err := db.DeleteCustomDomain(ctx, id, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		log.Error("failed to delete custom domain", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// isPlatformOwnedDomain reports whether the given normalized domain is owned by the platform
// and must therefore not be registrable as a custom domain: distr.sh (and subdomains), the
// instance's own app and registry hosts, and the CNAME target hosts.
func isPlatformOwnedDomain(domain string) bool {
	platformHosts := []string{"distr.sh", hostnameOf(env.Host()), hostnameOf(env.RegistryHost())}
	if target := env.CustomDomainAppCNAMETarget(); target != nil {
		platformHosts = append(platformHosts, *target)
	}
	if target := env.CustomDomainRegistryCNAMETarget(); target != nil {
		platformHosts = append(platformHosts, *target)
	}
	for _, host := range platformHosts {
		host = strings.ToLower(host)
		if host != "" && (domain == host || strings.HasSuffix(domain, "."+host)) {
			return true
		}
	}
	return false
}

// hostnameOf extracts the bare hostname from a host value that may contain a scheme and/or
// port (env.Host() is a base URL like "https://app.distr.sh", env.RegistryHost() may carry
// a port in development setups).
func hostnameOf(host string) string {
	if strings.Contains(host, "://") {
		if u, err := url.Parse(host); err == nil && u.Host != "" {
			return u.Hostname()
		}
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
