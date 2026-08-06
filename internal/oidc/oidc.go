package oidc

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
)

type Provider = types.OIDCProvider

const (
	ProviderGithub    = types.OIDCProviderGithub
	ProviderGoogle    = types.OIDCProviderGoogle
	ProviderMicrosoft = types.OIDCProviderMicrosoft
	ProviderGeneric   = types.OIDCProviderGeneric
)

// githubIssuer is a synthetic issuer, because GitHub is plain OAuth2 and does not issue
// an ID token that could provide one.
const githubIssuer = "https://github.com"

// Identity is the user identity as reported by an identity provider. Issuer and Subject
// identify the user at the provider and remain stable when the email address changes.
type Identity struct {
	Provider      Provider
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
}

// IdentityExtractorFunc turns a token response into the identity of the user it was issued for. The nonce
// is the one of the authorization request, and is empty for providers that issue no ID token to carry it.
type IdentityExtractorFunc func(ctx context.Context, token *oauth2.Token, nonce string) (Identity, error)

// NormalizeScopes accepts scopes as a list, or as comma or space separated values in any of its
// entries, and always requests openid first, because without it a provider issues no ID token.
func NormalizeScopes(values []string) []string {
	scopes := []string{oidc.ScopeOpenID}
	for _, value := range values {
		for _, scope := range strings.FieldsFunc(value, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
			if !slices.Contains(scopes, scope) {
				scopes = append(scopes, scope)
			}
		}
	}
	return scopes
}

func idTokenIdentityExtractor(
	provider Provider,
	oidcProvider *oidc.Provider,
	verifier *oidc.IDTokenVerifier,
) IdentityExtractorFunc {
	return func(ctx context.Context, token *oauth2.Token, nonce string) (Identity, error) {
		return identityFromIDToken(ctx, provider, oidcProvider, verifier, token, nonce)
	}
}

// identityFromIDToken verifies the ID token of a token response and reads the identity from its claims.
// The nonce is compared to the one of the authorization request, so an ID token cannot be replayed in
// another login. Providers that only expose the email on their userinfo endpoint are asked for it there.
func identityFromIDToken(
	ctx context.Context,
	provider Provider,
	oidcProvider *oidc.Provider,
	verifier *oidc.IDTokenVerifier,
	token *oauth2.Token,
	nonce string,
) (Identity, error) {
	idTokenStr, ok := token.Extra("id_token").(string)
	if !ok {
		return Identity{}, fmt.Errorf("id_token not found in token response")
	}
	idToken, err := verifier.Verify(ctx, idTokenStr)
	if err != nil {
		return Identity{}, fmt.Errorf("failed to verify id_token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		return Identity{}, fmt.Errorf("id_token nonce does not match the authorization request")
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("failed to parse id_token claims: %w", err)
	}

	identity := Identity{
		Provider:      provider,
		Issuer:        idToken.Issuer,
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified != nil && *claims.EmailVerified,
	}
	if identity.Email != "" {
		return identity, nil
	}

	userInfo, err := oidcProvider.UserInfo(ctx, oauth2.StaticTokenSource(token))
	if err != nil {
		return Identity{}, fmt.Errorf("id_token carries no email and userinfo failed: %w", err)
	}
	var userInfoClaims struct {
		EmailVerified *bool `json:"email_verified"`
	}
	if err := userInfo.Claims(&userInfoClaims); err != nil {
		return Identity{}, fmt.Errorf("failed to parse userinfo claims: %w", err)
	}
	if userInfo.Email == "" {
		return Identity{}, fmt.Errorf("neither the id_token nor userinfo carries an email address")
	}
	identity.Email = userInfo.Email
	identity.EmailVerified = userInfoClaims.EmailVerified != nil && *userInfoClaims.EmailVerified
	return identity, nil
}

type providerContext struct {
	oauth2Config      func(r *http.Request) *config
	identityExtractor IdentityExtractorFunc
	// nonceSupported is false for GitHub, which is plain OAuth2 and has no ID token to carry a nonce.
	nonceSupported bool
}

type OIDCer struct {
	providers map[Provider]*providerContext
}

type config struct {
	oauth2.Config
	pkceEnabled bool
}

func NewOIDCer(ctx context.Context, log *zap.Logger) (*OIDCer, error) {
	p := make(map[Provider]*providerContext)
	if env.OIDCGoogleEnabled() {
		log.Info("initializing google OIDC")
		googleProvider, err := oidc.NewProvider(ctx, "https://accounts.google.com")
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Google OIDC provider: %w", err)
		}
		googleOidcConfig := &oidc.Config{ClientID: *env.OIDCGoogleClientID()}
		googleVerifier := googleProvider.Verifier(googleOidcConfig)
		p[ProviderGoogle] = &providerContext{
			oauth2Config:      getGoogleOauth2Config,
			identityExtractor: idTokenIdentityExtractor(ProviderGoogle, googleProvider, googleVerifier),
			nonceSupported:    true,
		}
	}
	if env.OIDCMicrosoftEnabled() {
		log.Info("initializing microsoft OIDC")
		microsoftProvider, err := oidc.NewProvider(ctx,
			fmt.Sprintf("https://login.microsoftonline.com/%v/v2.0", *env.OIDCMicrosoftTenantID()))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Microsoft OIDC provider: %w", err)
		}
		microsoftOidcConfig := &oidc.Config{ClientID: *env.OIDCMicrosoftClientID()}
		microsoftVerifier := microsoftProvider.Verifier(microsoftOidcConfig)
		p[ProviderMicrosoft] = &providerContext{
			oauth2Config: getMicrosoftOauth2Config,
			identityExtractor: idTokenIdentityExtractor(
				ProviderMicrosoft, microsoftProvider, microsoftVerifier),
			nonceSupported: true,
		}
	}
	if env.OIDCGithubEnabled() {
		log.Info("initializing github OIDC")
		p[ProviderGithub] = &providerContext{
			oauth2Config:      getGithubOauth2Config,
			identityExtractor: getIdentityFromGithubAccessToken,
		}
	}
	if env.OIDCGenericEnabled() {
		log.Info("initializing generic OIDC")
		genericProvider, err := oidc.NewProvider(ctx, *env.OIDCGenericIssuer())
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Generic OIDC provider: %w", err)
		}
		genericOidcConfig := &oidc.Config{ClientID: *env.OIDCGenericClientID()}
		genericVerifier := genericProvider.Verifier(genericOidcConfig)
		p[ProviderGeneric] = &providerContext{
			oauth2Config: func(r *http.Request) *config {
				return &config{
					Config: oauth2.Config{
						ClientID:     *env.OIDCGenericClientID(),
						ClientSecret: *env.OIDCGenericClientSecret(),
						RedirectURL:  getRedirectURL(r, ProviderGeneric),
						Endpoint:     genericProvider.Endpoint(),
						Scopes:       NormalizeScopes([]string{*env.OIDCGenericScopes()}),
					},
					pkceEnabled: env.OIDCGenericPKCEEnabled(),
				}
			},
			identityExtractor: idTokenIdentityExtractor(ProviderGeneric, genericProvider, genericVerifier),
			nonceSupported:    true,
		}
	}
	return &OIDCer{providers: p}, nil
}

func getGoogleOauth2Config(r *http.Request) *config {
	return &config{
		Config: oauth2.Config{
			ClientID:     *env.OIDCGoogleClientID(),
			ClientSecret: *env.OIDCGoogleClientSecret(),
			RedirectURL:  getRedirectURL(r, ProviderGoogle),
			Endpoint:     google.Endpoint,
			Scopes:       []string{oidc.ScopeOpenID, "email"},
		},
		pkceEnabled: true,
	}
}

func getMicrosoftOauth2Config(r *http.Request) *config {
	return &config{
		Config: oauth2.Config{
			ClientID:     *env.OIDCMicrosoftClientID(),
			ClientSecret: *env.OIDCMicrosoftClientSecret(),
			RedirectURL:  getRedirectURL(r, ProviderMicrosoft),
			Endpoint:     microsoft.AzureADEndpoint(*env.OIDCMicrosoftTenantID()),
			Scopes:       []string{oidc.ScopeOpenID, "email"},
		},
		pkceEnabled: true,
	}
}

func getGithubOauth2Config(r *http.Request) *config {
	return &config{
		Config: oauth2.Config{
			ClientID:     *env.OIDCGithubClientID(),
			ClientSecret: *env.OIDCGithubClientSecret(),
			RedirectURL:  getRedirectURL(r, ProviderGithub),
			Endpoint:     github.Endpoint,
			Scopes:       []string{oidc.ScopeOpenID, "email", "user:email"},
		},
		pkceEnabled: true,
	}
}

// GetIdentityForCode exchanges the code for a token and extracts the user's identity at the provider.
func (o *OIDCer) GetIdentityForCode(
	ctx context.Context, provider Provider, code, pkceVerifier, nonce string, r *http.Request,
) (Identity, error) {
	prov := o.providers[provider]
	if prov == nil || prov.oauth2Config == nil {
		return Identity{}, fmt.Errorf("OIDC provider not configured: %s", provider)
	}
	c := prov.oauth2Config(r)
	var opts []oauth2.AuthCodeOption
	if c.pkceEnabled {
		opts = append(opts, oauth2.VerifierOption(pkceVerifier))
	}
	token, err := c.Exchange(ctx, code, opts...)
	if err != nil {
		return Identity{}, fmt.Errorf("token exchange failed: %w", err)
	}

	if !prov.nonceSupported {
		nonce = ""
	}
	return prov.identityExtractor(ctx, token, nonce)
}

func getIdentityFromGithubAccessToken(ctx context.Context, token *oauth2.Token, _ string) (Identity, error) {
	accessToken, ok := token.Extra("access_token").(string)
	if !ok {
		return Identity{}, fmt.Errorf("access_token not found in token response")
	}

	var user struct {
		ID int64 `json:"id"`
	}
	if err := getGithubResource(ctx, accessToken, "https://api.github.com/user", &user); err != nil {
		return Identity{}, err
	} else if user.ID == 0 {
		return Identity{}, fmt.Errorf("no user id returned by GitHub")
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getGithubResource(ctx, accessToken, "https://api.github.com/user/emails", &emails); err != nil {
		return Identity{}, err
	}

	for _, email := range emails {
		if email.Primary && email.Verified {
			return Identity{
				Provider: ProviderGithub,
				Issuer:   githubIssuer,
				// the numeric user id is stable across username and email changes
				Subject:       strconv.FormatInt(user.ID, 10),
				Email:         email.Email,
				EmailVerified: true,
			}, nil
		}
	}
	return Identity{}, fmt.Errorf("no primary verified email found")
}

func getGithubResource(ctx context.Context, accessToken, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch %v: %v", url, resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// GetAuthCodeURL returns the OIDC provider's AuthCodeURL for the given state and provider.
func (o *OIDCer) GetAuthCodeURL(
	r *http.Request, provider Provider, state, pkceVerifier, nonce string,
) (string, error) {
	prov := o.providers[provider]
	if prov == nil || prov.oauth2Config == nil {
		return "", fmt.Errorf("OIDC provider not configured: %s", provider)
	}
	c := prov.oauth2Config(r)
	var opts []oauth2.AuthCodeOption
	if prov.nonceSupported {
		opts = append(opts, oidc.Nonce(nonce))
	}
	if c.pkceEnabled {
		opts = append(opts, oauth2.S256ChallengeOption(pkceVerifier))
	}
	return c.AuthCodeURL(state, opts...), nil
}
