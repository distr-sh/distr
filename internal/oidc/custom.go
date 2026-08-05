package oidc

import (
	"context"
	"crypto/subtle"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/distr-sh/distr/internal/types"
	"golang.org/x/oauth2"
)

type CustomProvider struct {
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth2Config oauth2.Config
	pkceEnabled  bool
}

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

func (p *CustomProvider) AuthCodeURL(state, nonce, pkceVerifier string) string {
	opts := []oauth2.AuthCodeOption{oidc.Nonce(nonce)}
	if p.pkceEnabled {
		opts = append(opts, oauth2.S256ChallengeOption(pkceVerifier))
	}
	return p.oauth2Config.AuthCodeURL(state, opts...)
}

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
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
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
