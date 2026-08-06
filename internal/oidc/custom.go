package oidc

import (
	"context"
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

func (p *CustomProvider) AuthCodeURL(state, pkceVerifier, nonce string) string {
	opts := []oauth2.AuthCodeOption{oidc.Nonce(nonce)}
	if p.pkceEnabled {
		opts = append(opts, oauth2.S256ChallengeOption(pkceVerifier))
	}
	return p.oauth2Config.AuthCodeURL(state, opts...)
}

func (p *CustomProvider) IdentityForCode(ctx context.Context, code, pkceVerifier, nonce string) (Identity, error) {
	ctx = RestrictedClientContext(ctx)

	var opts []oauth2.AuthCodeOption
	if p.pkceEnabled {
		opts = append(opts, oauth2.VerifierOption(pkceVerifier))
	}
	token, err := p.oauth2Config.Exchange(ctx, code, opts...)
	if err != nil {
		return Identity{}, fmt.Errorf("token exchange failed: %w", err)
	}
	return identityFromIDToken(ctx, types.OIDCProviderCustom, p.provider, p.verifier, token, nonce)
}
