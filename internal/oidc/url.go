package oidc

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/types"
)

func getRedirectURL(r *http.Request, provider types.OIDCProvider) string {
	return fmt.Sprintf("%v/api/v1/auth/oidc/%v/callback", handlerutil.GetRequestSchemeAndHost(r), provider)
}

// CustomLoginPath identifies a custom provider by the slug of its organization and its own slug,
// rather than by its ID: the identity provider needs the callback URL registered as a redirect URI,
// and an administrator has to be able to read and type it.
func CustomLoginPath(organizationSlug, slug string) string {
	return fmt.Sprintf("/api/v1/auth/oidc/custom/%v/%v", organizationSlug, slug)
}

func CustomRedirectURL(r *http.Request, organizationSlug, slug string) string {
	return fmt.Sprintf("%v%v/callback",
		handlerutil.GetRequestSchemeAndHost(r), CustomLoginPath(organizationSlug, slug))
}

// CustomCallbackURL is the callback URL of a custom provider on the given app domain, to be shown so
// that an administrator can register it as a redirect URI. It uses the scheme of this instance, the
// same one CustomRedirectURL derives from the request, so that the URL shown is the redirect_uri the
// login actually sends. An empty string is returned while a part of the URL is still missing.
func CustomCallbackURL(domain string, organizationSlug *string, slug string) string {
	if domain == "" || organizationSlug == nil || *organizationSlug == "" || slug == "" {
		return ""
	}
	return fmt.Sprintf("%v://%v%v/callback", env.HostScheme(), domain, CustomLoginPath(*organizationSlug, slug))
}
