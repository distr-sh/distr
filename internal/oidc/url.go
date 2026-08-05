package oidc

import (
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/handlerutil"
)

func getRedirectURL(r *http.Request, provider Provider) string {
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
