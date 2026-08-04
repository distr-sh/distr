package oidc

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/distr-sh/distr/internal/handlerutil"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// CustomProvider is a provider configured by an organization, ready to run one authorization step.
// It is built per request rather than cached, so an edited configuration takes effect immediately;
// the cost is one discovery request per login step, which is irrelevant at login frequency.
type CustomProvider struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	pkceEnabled  bool
}

// ProviderForConfiguration resolves the configuration's issuer and prepares the authorization
// request. redirectURL must be the callback on the host the login was started from, because the
// identity provider binds the authorization code to it.
func ProviderForConfiguration(
	ctx context.Context,
	configuration types.CustomOIDCConfiguration,
	redirectURL string,
) (*CustomProvider, error) {
	discovered, err := Discover(ctx, configuration.Issuer)
	if err != nil {
		return nil, err
	}
	pkceEnabled := discovered.PKCESupported
	if configuration.PKCEEnabled != nil {
		pkceEnabled = *configuration.PKCEEnabled
	}
	return &CustomProvider{
		provider: discovered.Provider,
		verifier: discovered.Provider.Verifier(&oidc.Config{ClientID: configuration.ClientID}),
		oauth2Config: oauth2.Config{
			ClientID:     configuration.ClientID,
			ClientSecret: configuration.ClientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     discovered.Provider.Endpoint(),
			Scopes:       configuration.Scopes,
		},
		pkceEnabled: pkceEnabled,
	}, nil
}

// CustomRedirectURL is the callback of an organization configuration on the host of the given
// request. The configuration is only offered on its own custom domain, so the host is stable.
func CustomRedirectURL(r *http.Request, configurationID uuid.UUID) string {
	return fmt.Sprintf("%v/api/v1/auth/oidc/custom/%v/callback",
		handlerutil.GetRequestSchemeAndHost(r), configurationID)
}

func (p *CustomProvider) AuthCodeURL(state, nonce, pkceVerifier string) string {
	opts := []oauth2.AuthCodeOption{oidc.Nonce(nonce)}
	if p.pkceEnabled {
		opts = append(opts, oauth2.S256ChallengeOption(pkceVerifier))
	}
	return p.oauth2Config.AuthCodeURL(state, opts...)
}

// IdentityForCode exchanges the authorization code and returns the verified identity. The nonce is
// the one that was sent with the authorization request: checking it ties the ID token to this
// login attempt, so a token obtained elsewhere cannot be replayed here.
func (p *CustomProvider) IdentityForCode(ctx context.Context, code, pkceVerifier, nonce string) (Identity, error) {
	ctx = oidc.ClientContext(ctx, discoveryHTTPClient())

	var opts []oauth2.AuthCodeOption
	if p.pkceEnabled {
		opts = append(opts, oauth2.VerifierOption(pkceVerifier))
	}
	token, err := p.oauth2Config.Exchange(ctx, code, opts...)
	if err != nil {
		return Identity{}, fmt.Errorf("token exchange failed: %w", err)
	}

	idTokenStr, ok := token.Extra("id_token").(string)
	if !ok {
		return Identity{}, fmt.Errorf("id_token not found in token response")
	}
	idToken, err := p.verifier.Verify(ctx, idTokenStr)
	if err != nil {
		return Identity{}, fmt.Errorf("failed to verify id_token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		return Identity{}, fmt.Errorf("id_token nonce does not match the authorization request")
	}

	var claims struct {
		Email string `json:"email"`
		// A missing email_verified claim means "unknown", not "unverified" — Entra ID usually
		// omits it. It only ever decides whether the address counts as verified in Distr.
		EmailVerified *bool `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("failed to parse id_token claims: %w", err)
	}

	identity := Identity{
		Provider:      types.OIDCProviderCustom,
		Issuer:        idToken.Issuer,
		Subject:       idToken.Subject,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified != nil && *claims.EmailVerified,
	}
	if identity.Email == "" {
		// Okta can be configured to leave the email out of the ID token, and Entra ID omits it
		// for accounts without a mail attribute. UserInfo goes through the provider, which
		// cross-checks that the response belongs to the same subject.
		userInfo, err := p.provider.UserInfo(ctx, oauth2.StaticTokenSource(token))
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
	}
	return identity, nil
}
