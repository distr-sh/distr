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

func redirectLoginToAppDomain(w http.ResponseWriter, r *http.Request, user types.UserAccount, token string) {
	if redirect := loginAppDomainRedirect(r.Context(), r, user, token); redirect != nil {
		http.Redirect(w, r, *redirect, http.StatusFound)
		return
	}
	http.Redirect(w, r,
		fmt.Sprintf("%v/login?jwt=%v", handlerutil.GetRequestSchemeAndHost(r), token),
		http.StatusFound)
}
