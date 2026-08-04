package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/subscription"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/userauth"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

const (
	redirectToLoginOIDCFailed               = "/login?reason=oidc-failed"
	redirectToLoginOIDCRegistrationDisabled = "/login?reason=oidc-registration-disabled"
	redirectToLoginOIDCUnavailable          = "/login?reason=oidc-unavailable"
	// redirectToLoginOIDCNoAccount is used when the identity provider authenticated somebody who
	// has no account in the organization and must not be provisioned one.
	redirectToLoginOIDCNoAccount = "/login?reason=oidc-no-account"
	// redirectToLoginOIDCUserLimit is used when provisioning would exceed the billed user seats,
	// which are never raised automatically.
	redirectToLoginOIDCUserLimit = "/login?reason=oidc-user-limit"
	// redirectToLoginOIDCNotExclusive is used when the account is a member of another organization
	// as well, which no organization's own identity provider may authenticate.
	redirectToLoginOIDCNotExclusive = "/login?reason=oidc-account-not-exclusive"
)

func AuthOIDCRouter(r chiopenapi.Router) {
	type OIDCRequest struct {
		OIDCProvider string `path:"oidcProvider"`
	}
	type CustomOIDCRequest struct {
		CustomOIDCConfigurationID uuid.UUID `path:"customOidcConfigurationId"`
	}

	r.Get("/custom/{customOidcConfigurationId}", authLoginCustomOidcHandler).
		With(option.Request(CustomOIDCRequest{}))
	r.Get("/custom/{customOidcConfigurationId}/callback", authLoginCustomOidcCallbackHandler).
		With(option.Request(CustomOIDCRequest{}))
	r.Get("/{oidcProvider}", authLoginOidcHandler).
		With(option.Request(OIDCRequest{}))
	r.Get("/{oidcProvider}/callback", authLoginOidcCallbackHandler).
		With(option.Request(OIDCRequest{}))
}

func authLoginOidcHandler(w http.ResponseWriter, r *http.Request) {
	provider := oidc.Provider(r.PathValue("oidcProvider"))
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	// The instance-scoped providers belong to the platform, not to the organization owning the domain, so they are
	// not offered on a self-service custom domain. Hiding the buttons is not enough, since the auth routes are
	// reachable directly. Only the initiation is gated: the IdP binds the redirect_uri to the initiating host.
	// The same resolution backs the portal response, so the buttons and this gate can never disagree.
	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if !host.instanceAuthAllowed() {
		log.Info("rejecting instance OIDC login on custom domain",
			zap.String("provider", string(provider)), zap.String("host", r.Host))
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return
	} else if err != nil {
		// Fail closed: an unresolved host may well be a custom domain.
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("could not resolve host for OIDC login", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	if state, err := db.CreateOIDCState(ctx, nil); err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("OIDC state creation failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	} else {
		oidcer := internalctx.GetOIDCer(ctx)
		redirectURL, err := oidcer.GetAuthCodeURL(r, provider, state.ID.String(), state.PKCECodeVerifier)
		if err != nil {
			http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
			return
		}
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

func authLoginOidcCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	state, ok := consumeOIDCState(w, r, nil)
	if !ok {
		return
	}

	provider := oidc.Provider(r.PathValue("oidcProvider"))
	log = log.With(zap.String("provider", string(provider)))

	code, ok := oidcCallbackCode(w, r, log)
	if !ok {
		return
	}

	oidcer := internalctx.GetOIDCer(ctx)
	identity, err := oidcer.GetIdentityForCode(ctx, provider, code, state.PKCECodeVerifier, r)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("OIDC identity extraction failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		user, err := resolveOIDCUser(ctx, log, identity)
		if err != nil {
			return err
		}
		if user == nil {
			http.Redirect(w, r, redirectToLoginOIDCRegistrationDisabled, http.StatusFound)
			return nil
		}
		log = log.With(zap.Any("userId", user.ID))

		if user.EmailVerifiedAt == nil && identity.EmailVerified {
			if err = db.UpdateUserAccountEmailVerified(ctx, user); err != nil {
				return err
			}
		}
		if tokenString, err := userauth.GenerateLoginToken(ctx, *user); err != nil {
			return fmt.Errorf("token creation failed: %w", err)
		} else if err = db.UpdateUserAccountLastLoggedIn(ctx, user.ID); err != nil {
			return err
		} else {
			// An organization with its own app domain is sent there, so its users end up on the
			// host that offers their organization's login methods.
			redirectLoginToAppDomain(w, r, *user, tokenString)
			return nil
		}
	})
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Warn("user login failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
	}
}

// authLoginCustomOidcHandler starts a login through a provider configured by an organization. The
// provider is only reachable on the custom domain it is bound to: the request host decides which
// configurations exist at all, so a provider cannot be used from the default host or from another
// organization's domain.
func authLoginCustomOidcHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	configuration, ok := resolveCustomOIDCConfigurationForHost(w, r)
	if !ok {
		return
	}
	log = log.With(zap.Any("customOidcConfigurationId", configuration.ID))

	provider, err := oidc.ProviderForConfiguration(ctx, *configuration, oidc.CustomRedirectURL(r, configuration.ID))
	if err != nil {
		// A broken configuration is the organization's to fix, so this is not reported to Sentry.
		log.Warn("could not build custom OIDC provider", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	state, err := db.CreateOIDCState(ctx, &configuration.ID)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("OIDC state creation failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}
	http.Redirect(w, r, provider.AuthCodeURL(state.ID.String(), state.Nonce, state.PKCECodeVerifier),
		http.StatusFound)
}

func authLoginCustomOidcCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	configuration, ok := resolveCustomOIDCConfigurationForHost(w, r)
	if !ok {
		return
	}
	log = log.With(zap.Any("customOidcConfigurationId", configuration.ID))

	state, ok := consumeOIDCState(w, r, &configuration.ID)
	if !ok {
		return
	}
	code, ok := oidcCallbackCode(w, r, log)
	if !ok {
		return
	}

	provider, err := oidc.ProviderForConfiguration(ctx, *configuration, oidc.CustomRedirectURL(r, configuration.ID))
	if err != nil {
		log.Warn("could not build custom OIDC provider", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}
	identity, err := provider.IdentityForCode(ctx, code, state.PKCECodeVerifier, state.Nonce)
	if err != nil {
		log.Warn("custom OIDC identity extraction failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		user, failure, err := resolveCustomOIDCUser(ctx, log, *configuration, identity)
		if err != nil {
			return err
		}
		if user == nil {
			log.Info("custom OIDC login refused", zap.String("redirect", failure))
			http.Redirect(w, r, failure, http.StatusFound)
			return nil
		}
		log = log.With(zap.Any("userId", user.ID))

		if user.EmailVerifiedAt == nil && identity.EmailVerified {
			if err := db.UpdateUserAccountEmailVerified(ctx, user); err != nil {
				return err
			}
		}
		// The token is pinned to the configuration's organization: the login says nothing about any
		// other organization, and under the exclusivity rule there is no other one anyway.
		tokenString, err := userauth.GenerateLoginTokenForOrganization(ctx, *user, configuration.OrganizationID)
		if err != nil {
			return fmt.Errorf("token creation failed: %w", err)
		}
		if err := db.UpdateUserAccountLastLoggedIn(ctx, user.ID); err != nil {
			return err
		}
		http.Redirect(w, r,
			fmt.Sprintf("%v/login?jwt=%v", handlerutil.GetRequestSchemeAndHost(r), tokenString),
			http.StatusFound)
		return nil
	})
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Warn("custom OIDC login failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
	}
}

// resolveCustomOIDCConfigurationForHost loads the configuration named in the path and verifies that
// it is offered on the request host. Both the initiation and the callback go through this, so a
// configuration is unusable outside its own domain even when its id is known.
func resolveCustomOIDCConfigurationForHost(
	w http.ResponseWriter,
	r *http.Request,
) (*types.CustomOIDCConfiguration, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	id, err := uuid.Parse(r.PathValue("customOidcConfigurationId"))
	if err != nil {
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return nil, false
	}

	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("could not resolve host for custom OIDC login", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return nil, false
	}

	configuration, err := db.GetCustomOIDCConfiguration(ctx, id)
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return nil, false
	} else if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("could not get custom OIDC configuration", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return nil, false
	}

	if !configuration.Enabled ||
		host.customDomainRow == nil || host.customDomainRow.ID != configuration.CustomDomainID {
		log.Info("rejecting custom OIDC login on foreign host",
			zap.Any("customOidcConfigurationId", id), zap.String("host", r.Host))
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return nil, false
	}

	organization, err := db.GetOrganizationByID(ctx, configuration.OrganizationID)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("could not get organization of custom OIDC configuration", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return nil, false
	}
	// The feature is checked on every login, not only when the configuration is saved, so a
	// provider stops working when the organization loses the plan that granted it.
	if !organization.HasFeature(types.FeatureCustomOidcProviders) {
		log.Info("rejecting custom OIDC login without the feature",
			zap.Any("organizationId", organization.ID))
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return nil, false
	}
	return configuration, true
}

// resolveCustomOIDCUser returns the account the identity belongs to, or the login page the refused
// login is sent to. A configuration may only ever authenticate an account that belongs to its
// organization and to no other one: an account with a membership elsewhere would turn the
// organization's provider into a way into that other organization.
func resolveCustomOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	configuration types.CustomOIDCConfiguration,
	identity oidc.Identity,
) (*types.UserAccount, string, error) {
	user, existingIdentity, err := db.GetUserAccountWithOIDCIdentity(
		ctx, &configuration.ID, identity.Issuer, identity.Subject)
	if err == nil {
		if exclusive, err := isExclusiveToOrganization(ctx, *user, configuration.OrganizationID); err != nil {
			return nil, "", err
		} else if !exclusive {
			return nil, redirectToLoginOIDCNotExclusive, nil
		}
		// The user account email is authoritative and is never overwritten with the one from the
		// identity provider, which is only kept on the identity for display.
		return user, "", db.UpdateUserAccountOIDCIdentityOnLogin(ctx, existingIdentity.ID, new(identity.Email))
	} else if !errors.Is(err, apierrors.ErrNotFound) {
		return nil, "", err
	}

	user, err = db.GetUserAccountByEmail(ctx, identity.Email)
	if errors.Is(err, apierrors.ErrNotFound) {
		if user, failure, err := provisionCustomOIDCUser(ctx, log, configuration, identity); err != nil {
			return nil, "", err
		} else if user == nil {
			return nil, failure, nil
		} else {
			return user, "", linkCustomOIDCIdentity(ctx, configuration, identity, *user)
		}
	} else if err != nil {
		return nil, "", err
	}

	// Linking an existing account by email is the only merge that ever happens, and only for an
	// account that is already a member of this organization and of no other: an invited user, or
	// one who signed up before the provider was configured.
	if exclusive, err := isExclusiveToOrganization(ctx, *user, configuration.OrganizationID); err != nil {
		return nil, "", err
	} else if !exclusive {
		return nil, redirectToLoginOIDCNotExclusive, nil
	}
	if member, err := isOrganizationMember(ctx, *user, configuration.OrganizationID); err != nil {
		return nil, "", err
	} else if !member {
		return nil, redirectToLoginOIDCNoAccount, nil
	}
	log.Info("linking custom OIDC identity to existing user account matched by email",
		zap.Any("userId", user.ID))
	return user, "", linkCustomOIDCIdentity(ctx, configuration, identity, *user)
}

// provisionCustomOIDCUser creates an account for an email address that has none yet. It returns
// (nil, redirect, nil) when the configuration does not allow it, the email domain is not allowed, or
// the organization has no seat left — seats are never raised automatically.
func provisionCustomOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	configuration types.CustomOIDCConfiguration,
	identity oidc.Identity,
) (*types.UserAccount, string, error) {
	if !configuration.CreateUnknownUsers {
		return nil, redirectToLoginOIDCNoAccount, nil
	}
	if !emailDomainAllowed(configuration, identity.Email) {
		log.Info("rejecting custom OIDC login for a disallowed email domain")
		return nil, redirectToLoginOIDCNoAccount, nil
	}

	organization, err := db.GetOrganizationByID(ctx, configuration.OrganizationID)
	if err != nil {
		return nil, "", err
	}
	if limitReached, err := subscription.IsBillableUserAccountLimitReached(ctx, *organization); err != nil {
		return nil, "", err
	} else if limitReached {
		log.Info("rejecting custom OIDC login, user account limit reached",
			zap.Any("organizationId", organization.ID))
		return nil, redirectToLoginOIDCUserLimit, nil
	}

	user := types.UserAccount{Email: identity.Email}
	if identity.EmailVerified {
		user.EmailVerifiedAt = new(time.Now())
	}
	if err := db.CreateUserAccount(ctx, &user); err != nil {
		return nil, "", err
	}
	if err := db.CreateUserAccountOrganizationAssignment(
		ctx, user.ID, configuration.OrganizationID, configuration.DefaultUserRole, nil, nil); err != nil {
		return nil, "", err
	}
	log.Info("provisioned user account for custom OIDC identity", zap.Any("userId", user.ID))
	return &user, "", nil
}

func linkCustomOIDCIdentity(
	ctx context.Context,
	configuration types.CustomOIDCConfiguration,
	identity oidc.Identity,
	user types.UserAccount,
) error {
	newIdentity := types.UserAccountOIDCIdentity{
		UserAccountID:             user.ID,
		Provider:                  types.OIDCProviderCustom,
		Issuer:                    identity.Issuer,
		Subject:                   identity.Subject,
		Email:                     new(identity.Email),
		CustomOIDCConfigurationID: new(configuration.ID),
	}
	return db.CreateUserAccountOIDCIdentity(ctx, &newIdentity)
}

// isExclusiveToOrganization reports whether the account may be authenticated by a provider of the
// given organization: it must not be a member of any other organization, and it must not be a super
// admin, whose account reaches every organization on the instance.
func isExclusiveToOrganization(ctx context.Context, user types.UserAccount, orgID uuid.UUID) (bool, error) {
	if user.IsSuperAdmin {
		return false, nil
	}
	count, err := db.CountUserAccountOrganizationsExcept(ctx, user.ID, orgID)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

func isOrganizationMember(ctx context.Context, user types.UserAccount, orgID uuid.UUID) (bool, error) {
	if _, err := db.GetUserAccountWithRole(ctx, user.ID, orgID, nil, nil); errors.Is(err, apierrors.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

// emailDomainAllowed reports whether an account may be provisioned for the given email address.
// An empty allowlist allows nothing: a provisioned account joins the organization's own team, and
// which addresses may do that is a decision the organization has to have made. The API and a table
// constraint both refuse such a configuration, so this only ever guards a row that predates them.
func emailDomainAllowed(configuration types.CustomOIDCConfiguration, email string) bool {
	_, domain, found := strings.Cut(strings.ToLower(email), "@")
	return found && slices.Contains(configuration.AllowedEmailDomains, domain)
}

// resolveOIDCUser returns the user account the given identity provider identity belongs to.
// The identity is looked up first, so a login keeps working when the email address changed
// on either side. Accounts that predate the identity linking are matched by email and get
// their identity created on the fly. Returns (nil, nil) if the user would have to be
// registered but registration is administratively disabled.
func resolveOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	identity oidc.Identity,
) (*types.UserAccount, error) {
	user, existingIdentity, err := db.GetUserAccountWithOIDCIdentity(ctx, nil, identity.Issuer, identity.Subject)
	if err == nil {
		// The user account email is authoritative and is never overwritten with the one
		// from the identity provider, which is only kept on the identity for display.
		return user, db.UpdateUserAccountOIDCIdentityOnLogin(ctx, existingIdentity.ID, new(identity.Email))
	} else if !errors.Is(err, apierrors.ErrNotFound) {
		return nil, err
	}

	user, err = db.GetUserAccountByEmail(ctx, identity.Email)
	if errors.Is(err, apierrors.ErrNotFound) {
		if user, err = registerOIDCUser(ctx, identity.Email); err != nil {
			return nil, err
		} else if user == nil {
			return nil, nil
		}
		log.Info("registered new user account for OIDC identity", zap.Any("userId", user.ID))
	} else if err != nil {
		return nil, err
	} else {
		log.Info("linking OIDC identity to existing user account matched by email",
			zap.Any("userId", user.ID))
	}

	newIdentity := types.UserAccountOIDCIdentity{
		UserAccountID: user.ID,
		Provider:      identity.Provider,
		Issuer:        identity.Issuer,
		Subject:       identity.Subject,
		Email:         new(identity.Email),
	}
	if err := db.CreateUserAccountOIDCIdentity(ctx, &newIdentity); err != nil {
		return nil, err
	}
	return user, nil
}

// registerOIDCUser creates a new user account for an OIDC-authenticated user.
// The account is created without a password; the user can set one later via the
// password reset flow if they want to also sign in with email and password.
// Returns (nil, nil) if registration is administratively disabled.
func registerOIDCUser(ctx context.Context, email string) (*types.UserAccount, error) {
	if env.Registration() == env.RegistrationDisabled {
		return nil, nil
	}
	userAccount := types.UserAccount{
		Email:           email,
		EmailVerifiedAt: new(time.Now()),
	}
	if err := db.CreateUserAccountWithOrganization(ctx, &userAccount, &types.Organization{}); err != nil {
		return nil, fmt.Errorf("failed to create OIDC user: %w", err)
	}
	return &userAccount, nil
}

// oidcCallbackCode returns the authorization code from the callback, or redirects to the login page
// when the provider reported an error instead.
func oidcCallbackCode(w http.ResponseWriter, r *http.Request, log *zap.Logger) (string, bool) {
	if oidcError := r.URL.Query().Get("error"); oidcError != "" {
		log.Warn("OIDC provider returned error",
			zap.String("error", oidcError),
			zap.String("error_description", r.URL.Query().Get("error_description")))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return "", false
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Warn("OIDC callback missing code parameter")
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return "", false
	}
	return code, true
}

// consumeOIDCState redeems the state of the callback exactly once and verifies that it was created
// for the same provider the callback arrived at.
func consumeOIDCState(
	w http.ResponseWriter,
	r *http.Request,
	customOIDCConfigurationID *uuid.UUID,
) (db.OIDCState, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	state, err := verifyOIDCState(r, customOIDCConfigurationID)
	if err != nil {
		if !errors.Is(err, apierrors.ErrBadRequest) {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}
		log.Warn("could not verify OIDC state", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return db.OIDCState{}, false
	}
	return state, true
}

func verifyOIDCState(r *http.Request, customOIDCConfigurationID *uuid.UUID) (db.OIDCState, error) {
	id, err := uuid.Parse(r.URL.Query().Get("state"))
	if err != nil {
		return db.OIDCState{}, fmt.Errorf("%w: %w", apierrors.ErrBadRequest, err)
	}
	state, err := db.DeleteOIDCState(r.Context(), id)
	if err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			return db.OIDCState{}, apierrors.ErrBadRequest
		}
		return db.OIDCState{}, err
	}
	if state.Expired() {
		return db.OIDCState{}, fmt.Errorf("%w: got an OIDC state that is too old: %v, created_at: %v, now: %v",
			apierrors.ErrBadRequest, id, state.CreatedAt, time.Now().UTC())
	}
	// A code issued for one provider must not be redeemable at another one, which would let a
	// provider assert an identity for a flow it was never part of.
	if (state.CustomOIDCConfigurationID == nil) != (customOIDCConfigurationID == nil) ||
		(state.CustomOIDCConfigurationID != nil && *state.CustomOIDCConfigurationID != *customOIDCConfigurationID) {
		return db.OIDCState{}, fmt.Errorf("%w: OIDC state %v belongs to a different provider",
			apierrors.ErrBadRequest, id)
	}
	return state, nil
}
