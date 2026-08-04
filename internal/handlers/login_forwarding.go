package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/customdomains"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/userauth"
	"github.com/distr-sh/distr/internal/validation"
	"go.uber.org/zap"
)

// loginAppDomainRedirect returns where a login that succeeded on the instance's default host has to
// continue: the app domain of the user's primary organization, with the session token handed over
// the same way the OIDC callback does. It returns nil when the login already happened on a custom
// domain, when the organization has none, or when anything about the resolution fails — the login
// then simply completes on the host it was made on.
//
// The handover is safe because it happens after authentication: the account is already identified,
// so this reveals nothing to somebody who does not hold the token anyway.
func loginAppDomainRedirect(ctx context.Context, r *http.Request, user types.UserAccount, token string) *string {
	log := internalctx.GetLogger(ctx)

	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if err != nil {
		log.Warn("could not resolve host for login forwarding", zap.Error(err))
		return nil
	}
	if host.source != portalHostDefault {
		return nil
	}

	organization, err := userauth.PrimaryOrganization(ctx, user)
	if errors.Is(err, apierrors.ErrNotFound) {
		return nil
	} else if err != nil {
		log.Warn("could not resolve organization for login forwarding", zap.Error(err))
		return nil
	}
	domain := customdomains.AppDomain(ctx, organization.ID)
	if domain == nil {
		return nil
	}
	return new(fmt.Sprintf("https://%v/login?jwt=%v", *domain, token))
}

// redirectLoginToAppDomain completes a login that was performed through a browser redirect flow. It
// hands the token to the organization's app domain when there is one, and to the requested host
// otherwise.
func redirectLoginToAppDomain(w http.ResponseWriter, r *http.Request, user types.UserAccount, token string) {
	if redirect := loginAppDomainRedirect(r.Context(), r, user, token); redirect != nil {
		http.Redirect(w, r, *redirect, http.StatusFound)
		return
	}
	http.Redirect(w, r,
		fmt.Sprintf("%v/login?jwt=%v", handlerutil.GetRequestSchemeAndHost(r), token),
		http.StatusFound)
}
