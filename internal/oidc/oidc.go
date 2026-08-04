package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

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

type IdentityExtractorFunc func(context.Context, *oauth2.Token) (Identity, error)

func verifiedIdTokenIdentityExtractor(provider Provider, verifier *oidc.IDTokenVerifier) IdentityExtractorFunc {
	return func(ctx context.Context, token *oauth2.Token) (Identity, error) {
		idTokenStr, ok := token.Extra("id_token").(string)
		if !ok {
			return Identity{}, fmt.Errorf("id_token not found in token response")
		}
		idToken, err := verifier.Verify(ctx, idTokenStr)
		if err != nil {
			return Identity{}, fmt.Errorf("failed to verify id_token: %w", err)
		}
		var claims struct {
			Email         string `json:"email"`
			EmailVerified bool   `json:"email_verified"`
		}
		if err := idToken.Claims(&claims); err != nil {
			return Identity{}, fmt.Errorf("failed to parse id_token claims: %w", err)
		}
		return Identity{
			Provider:      provider,
			Issuer:        idToken.Issuer,
			Subject:       idToken.Subject,
			Email:         claims.Email,
			EmailVerified: claims.EmailVerified,
		}, nil
	}
}

type providerContext struct {
	oauth2Config      func(r *http.Request) *config
	identityExtractor IdentityExtractorFunc
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
			identityExtractor: verifiedIdTokenIdentityExtractor(ProviderGoogle, googleVerifier),
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
			oauth2Config:      getMicrosoftOauth2Config,
			identityExtractor: verifiedIdTokenIdentityExtractor(ProviderMicrosoft, microsoftVerifier),
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
						Scopes:       env.OIDCGenericScopes(),
					},
					pkceEnabled: env.OIDCGenericPKCEEnabled(),
				}
			},
			identityExtractor: verifiedIdTokenIdentityExtractor(ProviderGeneric, genericVerifier),
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
	ctx context.Context, provider Provider, code, pkceVerifier string, r *http.Request,
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

	return prov.identityExtractor(ctx, token)
}

func getIdentityFromGithubAccessToken(ctx context.Context, token *oauth2.Token) (Identity, error) {
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
func (o *OIDCer) GetAuthCodeURL(r *http.Request, provider Provider, state, pkceVerifier string) (string, error) {
	prov := o.providers[provider]
	if prov == nil || prov.oauth2Config == nil {
		return "", fmt.Errorf("OIDC provider not configured: %s", provider)
	}
	c := prov.oauth2Config(r)
	if c.pkceEnabled {
		return c.AuthCodeURL(state, oauth2.S256ChallengeOption(pkceVerifier)), nil
	}
	return c.AuthCodeURL(state), nil
}
