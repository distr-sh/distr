package authjwt

import (
	"maps"
	"sync"
	"time"

	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwt"
)

const (
	defaultTokenExpiration = 24 * time.Hour
)

const (
	UserNameKey          = "name"
	UserEmailKey         = "email"
	UserEmailVerifiedKey = "email_verified"
	UserRoleKey          = "role"
	UserImageURLKey      = "image_url"
	OrgIdKey             = "org"
	CustomerOrgIDKey     = "c_org"
	PartnerOrgIDKey      = "p_org"
	PasswordResetKey     = "password_reset"
	SuperAdminKey        = "is_super_admin"

	audienceUserValue  = "user"
	audienceAgentValue = "agent"
)

// JWTAuth is for generating/validating JWTs.
// Here we use symmetric encryption for now. This has the downside that the token can not be validated by clients,
// which should be OK for now.
//
// TODO: Maybe migrate to asymmetric encryption at some point.
var JWTAuth = sync.OnceValue(func() *jwtauth.JWTAuth {
	return jwtauth.New("HS256", env.JWTSecret(), nil)
})

func GenerateDefaultToken(user types.UserAccount, org types.OrganizationWithUserRole) (jwt.Token, string, error) {
	return generateUserToken(user, &org, defaultTokenExpiration, nil)
}

func GenerateResetToken(user types.UserAccount) (jwt.Token, string, error) {
	return generateUserToken(user, nil, env.ResetTokenValidDuration(), map[string]any{
		PasswordResetKey:     true,
		UserEmailVerifiedKey: true,
	})
}

func GenerateVerificationTokenValidFor(user types.UserAccount) (jwt.Token, string, error) {
	return generateUserToken(user, nil, env.InviteTokenValidDuration(), map[string]any{UserEmailVerifiedKey: true})
}

// GenerateInviteToken generates a token for accepting an invitation. When emailVerified is true (the invite link
// was delivered via email to the invitee's address) the resulting token carries a verified email claim, so the
// invitee does not need to verify their email separately. When false (e.g. the invite URL was handed out manually
// via the API response) the email_verified claim is left to the default, so the invitee still has to verify their
// email after accepting the invitation when verification is required, but is treated as verified when it is not.
func GenerateInviteToken(user types.UserAccount, emailVerified bool) (jwt.Token, string, error) {
	var extraClaims map[string]any
	if emailVerified {
		extraClaims = map[string]any{UserEmailVerifiedKey: true}
	}
	return generateUserToken(user, nil, env.InviteTokenValidDuration(), extraClaims)
}

func generateUserToken(
	user types.UserAccount,
	org *types.OrganizationWithUserRole,
	validFor time.Duration,
	extraClaims map[string]any,
) (jwt.Token, string, error) {
	now := time.Now()
	claims := map[string]any{
		jwt.IssuedAtKey:      now,
		jwt.NotBeforeKey:     now,
		jwt.ExpirationKey:    now.Add(validFor),
		jwt.SubjectKey:       user.ID.String(),
		jwt.AudienceKey:      audienceUserValue,
		UserNameKey:          user.Name,
		UserEmailKey:         user.Email,
		UserEmailVerifiedKey: !env.UserEmailVerificationRequired() || user.EmailVerifiedAt != nil,
	}
	if url := mapping.CreateImageURL(user.ImageID); url != nil {
		claims[UserImageURLKey] = *url
	}
	if user.IsSuperAdmin {
		claims[SuperAdminKey] = true
	}
	if org != nil {
		claims[OrgIdKey] = org.ID.String()
		if !user.IsSuperAdmin {
			claims[UserRoleKey] = org.UserRole
		}
		if org.CustomerOrganizationID != nil {
			claims[CustomerOrgIDKey] = org.CustomerOrganizationID.String()
		}
		if org.PartnerOrganizationID != nil {
			claims[PartnerOrgIDKey] = org.PartnerOrganizationID.String()
		}
	}
	maps.Copy(claims, extraClaims)
	return JWTAuth().Encode(claims)
}

func GenerateAgentTokenValidFor(targetID, orgID uuid.UUID, validFor time.Duration) (jwt.Token, string, error) {
	now := time.Now()
	claims := map[string]any{
		jwt.IssuedAtKey:   now,
		jwt.NotBeforeKey:  now,
		jwt.ExpirationKey: now.Add(validFor),
		jwt.SubjectKey:    targetID.String(),
		jwt.AudienceKey:   audienceAgentValue,
		OrgIdKey:          orgID.String(),
	}
	return JWTAuth().Encode(claims)
}
