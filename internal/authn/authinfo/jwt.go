package authinfo

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v4/jwt"
)

func FromUserJWT(token jwt.Token) (*SimpleAuthInfo, error) {
	var result SimpleAuthInfo
	result.rawToken = token

	if subjectStr, ok := token.Subject(); !ok {
		return nil, fmt.Errorf("%w: JWT subject missing", authn.ErrBadAuthentication)
	} else if userID, err := uuid.Parse(subjectStr); err != nil {
		return nil, fmt.Errorf("%w: JWT subject is invalid: %w", authn.ErrBadAuthentication, err)
	} else {
		result.userID = userID
	}

	if orgIDStr, err := jwt.Get[string](token, authjwt.OrgIdKey); err == nil {
		if orgID, err := uuid.Parse(orgIDStr); err != nil {
			return nil, fmt.Errorf("%w: JWT orgId is invalid: %w", authn.ErrBadAuthentication, err)
		} else {
			result.organizationID = &orgID
		}
	}

	if userRoleStr, err := jwt.Get[string](token, authjwt.UserRoleKey); err == nil {
		if userRole, err := types.ParseUserRole(userRoleStr); err != nil {
			return nil, fmt.Errorf("%w: JWT userRole is invalid: %w", authn.ErrBadAuthentication, err)
		} else {
			result.userRole = &userRole
		}
	}

	if partnerOrgIDStr, err := jwt.Get[string](token, authjwt.PartnerOrgIDKey); err == nil {
		if partnerOrgID, err := uuid.Parse(partnerOrgIDStr); err != nil {
			return nil, fmt.Errorf("%w: JWT partnerOrgId is invalid: %w", authn.ErrBadAuthentication, err)
		} else {
			result.partnerOrganizationID = &partnerOrgID
		}
	}

	result.userEmail, _ = jwt.Get[string](token, authjwt.UserEmailKey)
	result.emailVerified, _ = jwt.Get[bool](token, authjwt.UserEmailVerifiedKey)
	result.isSuperAdmin, _ = jwt.Get[bool](token, authjwt.SuperAdminKey)

	if scope, err := jwt.Get[string](token, authjwt.TokenScopeKey); err == nil {
		result.tokenScope = authjwt.TokenScope(scope)
	}

	// Only the presence of the claim is evaluated, and deliberately without reading its value: a claim
	// this server cannot make sense of must confine the session just the same.
	result.organizationScoped = token.Has(authjwt.CustomOIDCConfigurationIDKey)

	return &result, nil
}

func UserJWTAuthenticator() authn.Authenticator[jwt.Token, AuthInfo] {
	return authn.AuthenticatorFunc[jwt.Token, AuthInfo](
		func(ctx context.Context, token jwt.Token) (AuthInfo, error) {
			return FromUserJWT(token)
		},
	)
}

func FromAgentJWT(token jwt.Token) (*SimpleAgentAuthInfo, error) {
	var result SimpleAgentAuthInfo
	result.rawToken = token

	if subjectStr, ok := token.Subject(); !ok {
		return nil, fmt.Errorf("%w: JWT subject missing", authn.ErrBadAuthentication)
	} else if deploymentTargetID, err := uuid.Parse(subjectStr); err != nil {
		return nil, fmt.Errorf("%w: JWT subject is invalid: %w", authn.ErrBadAuthentication, err)
	} else {
		result.deploymentTargetID = deploymentTargetID
	}

	if orgIDStr, err := jwt.Get[string](token, authjwt.OrgIdKey); err == nil {
		if orgID, err := uuid.Parse(orgIDStr); err != nil {
			return nil, fmt.Errorf("%w: JWT orgId is invalid: %w", authn.ErrBadAuthentication, err)
		} else {
			result.organizationID = orgID
		}
	}

	return &result, nil
}

func AgentJWTAuthenticator() authn.Authenticator[jwt.Token, AgentAuthInfo] {
	return authn.AuthenticatorFunc[jwt.Token, AgentAuthInfo](
		func(ctx context.Context, token jwt.Token) (AgentAuthInfo, error) {
			return FromAgentJWT(token)
		},
	)
}
