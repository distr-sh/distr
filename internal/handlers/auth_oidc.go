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
	redirectToLoginOIDCExpired              = "/login?reason=oidc-expired"
	redirectToLoginOIDCRegistrationDisabled = "/login?reason=oidc-registration-disabled"
	redirectToLoginOIDCUnavailable          = "/login?reason=oidc-unavailable"
	redirectToLoginOIDCNoAccount            = "/login?reason=oidc-no-account"
	redirectToLoginOIDCUserLimit            = "/login?reason=oidc-user-limit"
	redirectToLoginOIDCOrgLimit             = "/login?reason=oidc-org-limit"
)

func AuthOIDCRouter(r chiopenapi.Router) {
	type OIDCRequest struct {
		OIDCProvider string `path:"oidcProvider"`
	}
	type CustomOIDCRequest struct {
		OrganizationSlug string `path:"organizationSlug"`
		ProviderSlug     string `path:"providerSlug"`
	}

	r.Get("/custom/{organizationSlug}/{providerSlug}", authLoginCustomOidcHandler).
		With(option.Request(CustomOIDCRequest{}))
	r.Get("/custom/{organizationSlug}/{providerSlug}/callback", authLoginCustomOidcCallbackHandler).
		With(option.Request(CustomOIDCRequest{}))
	r.Get("/{oidcProvider}", authLoginOidcHandler).
		With(option.Request(OIDCRequest{}))
	r.Get("/{oidcProvider}/callback", authLoginOidcCallbackHandler).
		With(option.Request(OIDCRequest{}))
}

func authLoginOidcHandler(w http.ResponseWriter, r *http.Request) {
	provider := types.OIDCProvider(r.PathValue("oidcProvider"))
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if !host.instanceAuthAllowed() {
		log.Info("rejecting instance OIDC login on custom domain",
			zap.String("provider", string(provider)), zap.String("host", r.Host))
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return
	} else if err != nil {
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
		redirectURL, err := oidcer.GetAuthCodeURL(r, provider, state.ID.String(), state.PKCECodeVerifier, state.Nonce)
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

	provider := types.OIDCProvider(r.PathValue("oidcProvider"))
	log = log.With(zap.String("provider", string(provider)))

	code, ok := oidcCallbackCode(w, r, log)
	if !ok {
		return
	}

	oidcer := internalctx.GetOIDCer(ctx)
	identity, err := oidcer.GetIdentityForCode(ctx, provider, code, state.PKCECodeVerifier, state.Nonce, r)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("OIDC identity extraction failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return
	}

	err = db.RunTx(ctx, func(ctx context.Context) error {
		user, redirect, err := resolveOIDCUser(ctx, log, identity)
		if err != nil {
			return err
		}

		if redirect != "" {
			http.Redirect(w, r, redirect, http.StatusFound)
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
			redirectLoginToAppDomain(w, r, *user, tokenString)
			return nil
		}
	})
	if errors.Is(err, subscription.ErrGlobalOrganizationLimitReached) {
		log.Info("rejecting OIDC login, global organization limit reached")
		http.Redirect(w, r, redirectToLoginOIDCOrgLimit, http.StatusFound)
	} else if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Warn("user login failed", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
	}
}

func authLoginCustomOidcHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	login, ok := resolveCustomOIDCConfigurationForHost(w, r)
	if !ok {
		return
	}
	configuration := login.configuration
	log = log.With(zap.Any("customOidcConfigurationId", configuration.ID))

	provider, err := oidc.ProviderForConfiguration(ctx, configuration, login.redirectURL)
	if err != nil {
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
	http.Redirect(w, r, provider.AuthCodeURL(state.ID.String(), state.PKCECodeVerifier, state.Nonce),
		http.StatusFound)
}

func authLoginCustomOidcCallbackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	login, ok := resolveCustomOIDCConfigurationForHost(w, r)
	if !ok {
		return
	}
	configuration := login.configuration
	log = log.With(zap.Any("customOidcConfigurationId", configuration.ID))

	state, ok := consumeOIDCState(w, r, &configuration.ID)
	if !ok {
		return
	}
	code, ok := oidcCallbackCode(w, r, log)
	if !ok {
		return
	}

	provider, err := oidc.ProviderForConfiguration(ctx, configuration, login.redirectURL)
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
		user, failure, err := resolveCustomOIDCUser(ctx, log, login, identity)
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
		tokenString, err := userauth.GenerateCustomOIDCLoginToken(ctx, *user, configuration)
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

type customOIDCLogin struct {
	configuration types.CustomOIDCConfiguration
	// domain is the CustomDomain the provider hangs off. Its customer organization is the scope every
	// membership check and every provisioned account is bound to.
	domain      types.CustomDomain
	redirectURL string
}

func (l customOIDCLogin) customerOrgID() *uuid.UUID {
	return l.domain.CustomerOrganizationID
}

// sharedCustomerPortal reports whether the provider is offered on the vendor's portal domain for all of
// its customers, where nothing in the OIDC response says which customer a new user would belong to.
func (l customOIDCLogin) sharedCustomerPortal() bool {
	return l.domain.Type == types.DomainTypeCustomerPortal && l.domain.CustomerOrganizationID == nil
}

func resolveCustomOIDCConfigurationForHost(
	w http.ResponseWriter,
	r *http.Request,
) (customOIDCLogin, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	organizationSlug := validation.NormalizeSlug(r.PathValue("organizationSlug"))
	providerSlug := validation.NormalizeSlug(r.PathValue("providerSlug"))

	host, err := resolvePortalHost(ctx, validation.NormalizeHostname(r.Host))
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("could not resolve host for custom OIDC login", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return customOIDCLogin{}, false
	}

	if host.customDomainRow == nil {
		log.Info("rejecting custom OIDC login on a host without a custom domain", zap.String("host", r.Host))
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return customOIDCLogin{}, false
	}

	configuration, err := db.GetCustomOIDCConfigurationForHost(
		ctx, host.customDomainRow.ID, organizationSlug, providerSlug)
	if errors.Is(err, apierrors.ErrNotFound) {
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return customOIDCLogin{}, false
	} else if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("could not get custom OIDC configuration", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return customOIDCLogin{}, false
	}

	if !configuration.Enabled {
		log.Info("rejecting custom OIDC login for a disabled provider",
			zap.Any("customOidcConfigurationId", configuration.ID))
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return customOIDCLogin{}, false
	}

	organization, err := db.GetOrganizationByID(ctx, configuration.OrganizationID)
	if err != nil {
		sentry.GetHubFromContext(ctx).CaptureException(err)
		log.Error("could not get organization of custom OIDC configuration", zap.Error(err))
		http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
		return customOIDCLogin{}, false
	}
	if !organization.HasFeature(types.FeatureCustomOidcProviders) {
		log.Info("rejecting custom OIDC login without the feature",
			zap.Any("organizationId", organization.ID))
		http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
		return customOIDCLogin{}, false
	}

	// A customer's provider stops working the moment the vendor revokes the feature, so a login is
	// refused for an identity that outlives the grant, the same way it is for one that outlives a
	// membership.
	if customerOrgID := host.customDomainRow.CustomerOrganizationID; customerOrgID != nil {
		customerOrganization, err := db.GetCustomerOrganizationByID(ctx, *customerOrgID)
		if err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			log.Error("could not get customer organization of custom OIDC configuration", zap.Error(err))
			http.Redirect(w, r, redirectToLoginOIDCFailed, http.StatusFound)
			return customOIDCLogin{}, false
		}
		if !customerOrganization.HasFeature(types.CustomerOrganizationFeatureOidcProviders) {
			log.Info("rejecting custom OIDC login without the customer feature",
				zap.Any("customerOrganizationId", *customerOrgID))
			http.Redirect(w, r, redirectToLoginOIDCUnavailable, http.StatusFound)
			return customOIDCLogin{}, false
		}
	}

	return customOIDCLogin{
		configuration: *configuration,
		domain:        *host.customDomainRow,
		redirectURL:   oidc.CustomRedirectURL(r, organizationSlug, configuration.Slug),
	}, true
}

func resolveCustomOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	login customOIDCLogin,
	identity oidc.Identity,
) (*types.UserAccount, string, error) {
	configuration := login.configuration
	user, existingIdentity, err := db.GetUserAccountWithOIDCIdentity(
		ctx, &configuration.ID, identity.Issuer, identity.Subject)
	if err == nil {
		if failure, err := checkCustomOIDCLoginAllowed(ctx, *user, login); err != nil {
			return nil, "", err
		} else if failure != "" {
			return nil, failure, nil
		}
		return user, "", db.UpdateUserAccountOIDCIdentityOnLogin(ctx, existingIdentity.ID, new(identity.Email))
	} else if !errors.Is(err, apierrors.ErrNotFound) {
		return nil, "", err
	}

	user, err = db.GetUserAccountByEmail(ctx, identity.Email)
	if errors.Is(err, apierrors.ErrNotFound) {
		if user, failure, err := provisionCustomOIDCUser(ctx, log, login, identity); err != nil {
			return nil, "", err
		} else if user == nil {
			return nil, failure, nil
		} else {
			return user, "", linkCustomOIDCIdentity(ctx, configuration, identity, *user)
		}
	} else if err != nil {
		return nil, "", err
	}

	if failure, err := admitExistingCustomOIDCUser(ctx, log, login, identity, *user); err != nil {
		return nil, "", err
	} else if failure != "" {
		return nil, failure, nil
	}
	log.Info("linking custom OIDC identity to existing user account matched by email",
		zap.Stringer("userId", user.ID))
	return user, "", linkCustomOIDCIdentity(ctx, configuration, identity, *user)
}

// admitExistingCustomOIDCUser gives an account that exists but is not a member of the provider's
// organization scope the same first sign-in an unknown account gets. Only a first sign-in: a revoked
// membership has to keep an already linked identity out for good.
func admitExistingCustomOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	login customOIDCLogin,
	identity oidc.Identity,
	user types.UserAccount,
) (string, error) {
	// Refused before provisioning, for the reason given in checkCustomOIDCLoginAllowed.
	if user.IsSuperAdmin {
		return redirectToLoginOIDCNoAccount, nil
	}

	if member, err := isOrganizationMember(ctx, user, login); err != nil {
		return "", err
	} else if member {
		return "", nil
	}

	if failure, err := checkCustomOIDCProvisioningAllowed(ctx, log, login, identity); err != nil || failure != "" {
		return failure, err
	}

	err := db.CreateUserAccountOrganizationAssignment(ctx, user.ID, login.configuration.OrganizationID,
		login.configuration.DefaultUserRole, login.customerOrgID(), nil)
	// One assignment exists per user and organization, so a user already in another scope of it cannot get
	// the provider's scope on top. Re-scoping instead would hand a customer's provider a member of the
	// vendor or of another customer.
	if errors.Is(err, apierrors.ErrAlreadyExists) {
		log.Info("rejecting custom OIDC login for a user assigned to another scope of the organization",
			zap.Stringer("userId", user.ID))
		return redirectToLoginOIDCNoAccount, nil
	} else if err != nil {
		return "", err
	}
	log.Info("assigned existing user account to the organization of a custom OIDC provider",
		zap.Stringer("userId", user.ID))
	return "", nil
}

// checkCustomOIDCLoginAllowed returns the login redirect to answer with when the user must not sign in
// through the organization's identity provider, or an empty string when the login may proceed. Membership
// is required for every login, so an identity that outlives the membership does not keep access alive.
// The session a login produces is confined to the organization of the provider (authjwt.GenerateCustomOIDCToken),
// which is what allows a member of several organizations to use one at all.
//
// A provider bound to one customer only ever matches a membership in that customer. Customer memberships
// all live on the vendor organization, so without the customer scope every customer's provider would
// authenticate every other customer's users.
func checkCustomOIDCLoginAllowed(
	ctx context.Context,
	user types.UserAccount,
	login customOIDCLogin,
) (string, error) {
	// A super admin is refused explicitly rather than by the membership check below, which they fail
	// already for not being assigned to any organization: their reach does not stop at an organization,
	// so confining the session confines nothing. The answer is the same as for an unknown account, so a
	// provider is not told which of the addresses it can assert belongs to a super admin.
	if user.IsSuperAdmin {
		return redirectToLoginOIDCNoAccount, nil
	}
	if member, err := isOrganizationMember(ctx, user, login); err != nil {
		return "", err
	} else if !member {
		return redirectToLoginOIDCNoAccount, nil
	}
	return "", nil
}

// checkCustomOIDCProvisioningAllowed returns the redirect to refuse a membership in the provider's
// organization scope with, or an empty string when the provider may hand one out.
func checkCustomOIDCProvisioningAllowed(
	ctx context.Context,
	log *zap.Logger,
	login customOIDCLogin,
	identity oidc.Identity,
) (string, error) {
	configuration := login.configuration
	if !configuration.CreateUnknownUsers {
		return redirectToLoginOIDCNoAccount, nil
	}
	// Refused at configuration time as well; this is the guard that survives a domain being re-pointed.
	if login.sharedCustomerPortal() {
		log.Info("rejecting custom OIDC provisioning on the shared customer portal domain",
			zap.Stringer("customOidcConfigurationId", configuration.ID))
		return redirectToLoginOIDCNoAccount, nil
	}
	if !emailDomainAllowed(configuration, identity.Email) {
		log.Info("rejecting custom OIDC login for a disallowed email domain")
		return redirectToLoginOIDCNoAccount, nil
	}

	organization, err := db.GetOrganizationByID(ctx, configuration.OrganizationID)
	if err != nil {
		return "", err
	}
	// A customer user counts against the per-customer limit, not the vendor's billable seats.
	customerOrgID := login.customerOrgID()
	var limitReached bool
	if customerOrgID != nil {
		customerOrganization, err := db.GetCustomerOrganizationByID(ctx, *customerOrgID)
		if err != nil {
			return "", err
		}
		limitReached, err = subscription.IsCustomerUserAccountLimitReached(*organization, *customerOrganization)
		if err != nil {
			return "", err
		}
	} else if limitReached, err = subscription.IsBillableUserAccountLimitReached(ctx, *organization); err != nil {
		return "", err
	}
	if limitReached {
		log.Info("rejecting custom OIDC login, user account limit reached",
			zap.Stringer("organizationId", organization.ID),
			zap.Stringer("customerOrganizationId", customerOrgID))
		return redirectToLoginOIDCUserLimit, nil
	}
	return "", nil
}

func provisionCustomOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	login customOIDCLogin,
	identity oidc.Identity,
) (*types.UserAccount, string, error) {
	configuration := login.configuration
	if failure, err := checkCustomOIDCProvisioningAllowed(ctx, log, login, identity); err != nil || failure != "" {
		return nil, failure, err
	}

	user := types.UserAccount{Email: identity.Email}
	if identity.EmailVerified {
		user.EmailVerifiedAt = new(time.Now())
	}
	if err := db.CreateUserAccount(ctx, &user); err != nil {
		return nil, "", err
	}
	if err := db.CreateUserAccountOrganizationAssignment(ctx, user.ID, configuration.OrganizationID,
		configuration.DefaultUserRole, login.customerOrgID(), nil); err != nil {
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

// isOrganizationMember also tells an app-domain membership apart from a shared-portal one:
// GetUserAccountWithRole's customerOrgID leaves the customer scope unfiltered when nil, which both pass.
func isOrganizationMember(ctx context.Context, user types.UserAccount, login customOIDCLogin) (bool, error) {
	member, err := db.GetUserAccountWithRole(
		ctx, user.ID, login.configuration.OrganizationID, login.customerOrgID(), nil)
	if errors.Is(err, apierrors.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if login.customerOrgID() != nil {
		return true, nil
	}
	if login.sharedCustomerPortal() {
		return member.CustomerOrganizationID != nil, nil
	}
	return member.CustomerOrganizationID == nil, nil
}

func emailDomainAllowed(configuration types.CustomOIDCConfiguration, email string) bool {
	_, domain, found := strings.Cut(strings.ToLower(email), "@")
	return found && slices.Contains(configuration.AllowedEmailDomains, domain)
}

// resolveOIDCUser returns the user account for the given OIDC identity, linking or registering it as needed.
// A non-empty redirect return means the login must be aborted with that redirect instead of proceeding.
func resolveOIDCUser(
	ctx context.Context,
	log *zap.Logger,
	identity oidc.Identity,
) (*types.UserAccount, string, error) {
	user, existingIdentity, err := db.GetUserAccountWithOIDCIdentity(ctx, nil, identity.Issuer, identity.Subject)
	if err == nil {
		return user, "", db.UpdateUserAccountOIDCIdentityOnLogin(ctx, existingIdentity.ID, new(identity.Email))
	} else if !errors.Is(err, apierrors.ErrNotFound) {
		return nil, "", err
	}

	user, err = db.GetUserAccountByEmail(ctx, identity.Email)
	if errors.Is(err, apierrors.ErrNotFound) {
		var redirect string
		if user, redirect, err = registerOIDCUser(ctx, log, identity.Email); err != nil || redirect != "" {
			return nil, redirect, err
		}
		log.Info("registered new user account for OIDC identity", zap.Any("userId", user.ID))
	} else if err != nil {
		return nil, "", err
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
		return nil, "", err
	}

	return user, "", nil
}

func registerOIDCUser(ctx context.Context, log *zap.Logger, email string) (*types.UserAccount, string, error) {
	if env.Registration() == env.RegistrationDisabled {
		return nil, redirectToLoginOIDCRegistrationDisabled, nil
	}

	if reached, err := subscription.IsGlobalOrganizationLimitReached(ctx); err != nil {
		return nil, "", err
	} else if reached {
		log.Info("rejecting OIDC registration, global organization limit reached")
		return nil, redirectToLoginOIDCOrgLimit, nil
	}

	userAccount := types.UserAccount{
		Email:           email,
		EmailVerifiedAt: new(time.Now()),
	}

	if err := db.CreateUserAccountWithOrganization(ctx, &userAccount, &types.Organization{}); err != nil {
		return nil, "", fmt.Errorf("failed to create OIDC user: %w", err)
	}

	return &userAccount, "", nil
}

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

func consumeOIDCState(
	w http.ResponseWriter,
	r *http.Request,
	customOIDCConfigurationID *uuid.UUID,
) (db.OIDCState, bool) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	state, err := verifyOIDCState(r, customOIDCConfigurationID)
	if err != nil {
		log.Warn("could not verify OIDC state", zap.Error(err))
		if errors.Is(err, apierrors.ErrBadRequest) {
			http.Redirect(w, r, redirectToLoginOIDCExpired, http.StatusFound)
			return db.OIDCState{}, false
		}
		sentry.GetHubFromContext(ctx).CaptureException(err)
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
	if (state.CustomOIDCConfigurationID == nil) != (customOIDCConfigurationID == nil) ||
		(state.CustomOIDCConfigurationID != nil && *state.CustomOIDCConfigurationID != *customOIDCConfigurationID) {
		return db.OIDCState{}, fmt.Errorf("%w: OIDC state %v belongs to a different provider",
			apierrors.ErrBadRequest, id)
	}
	return state, nil
}
