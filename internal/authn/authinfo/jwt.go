package authinfo

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/authjwt"
	"github.com/distr-sh/distr/internal/authn"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

func FromUserJWT(token jwt.Token) (*SimpleAuthInfo, error) {
	var result SimpleAuthInfo
	result.rawToken = token

	if subjectStr, ok := token.Subject(); !ok {
		return nil, fmt.Errorf("JWT subject missing")
	} else if userID, err := uuid.Parse(subjectStr); err != nil {
		return nil, fmt.Errorf("JWT subject is invalid: %w", err)
	} else {
		result.userID = userID
	}

	var orgIDStr string
	if err := token.Get(authjwt.OrgIdKey, &orgIDStr); err == nil {
		if orgID, err := uuid.Parse(orgIDStr); err != nil {
			return nil, fmt.Errorf("%w: JWT orgId is invalid: %w", authn.ErrBadAuthentication, err)
		} else {
			result.organizationID = &orgID
		}
	}

	var userRoleStr string
	if err := token.Get(authjwt.UserRoleKey, &userRoleStr); err == nil {
		if userRole, err := types.ParseUserRole(userRoleStr); err != nil {
			return nil, fmt.Errorf("%w: JWT userRole is invalid: %w", authn.ErrBadAuthentication, err)
		} else {
			result.userRole = &userRole
		}
	}

	_ = token.Get(authjwt.UserEmailKey, &result.userEmail)
	_ = token.Get(authjwt.UserEmailVerifiedKey, &result.emailVerified)
	_ = token.Get(authjwt.SuperAdminKey, &result.isSuperAdmin)

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
		return nil, fmt.Errorf("JWT subject missing")
	} else if deploymentTargetID, err := uuid.Parse(subjectStr); err != nil {
		return nil, fmt.Errorf("JWT subject is invalid: %w", err)
	} else {
		result.deploymentTargetID = deploymentTargetID
	}

	var orgIDStr string
	if err := token.Get(authjwt.OrgIdKey, &orgIDStr); err == nil {
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
