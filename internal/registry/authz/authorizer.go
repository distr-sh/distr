package authz

import (
	"context"
	"errors"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	"github.com/opencontainers/go-digest"
)

type Action string

const (
	ActionRead  Action = "read"
	ActionWrite Action = "write"
	ActionStat  Action = "stat"
)

type Authorizer interface {
	Authorize(ctx context.Context, name string, action Action) error
	AuthorizeReference(ctx context.Context, name string, reference string, action Action) error
	AuthorizeBlob(ctx context.Context, name string, digest digest.Digest, action Action) error
}

type authorizer struct{}

func NewAuthorizer() Authorizer {
	return &authorizer{}
}

// authorizeWrite verifies that the authenticated principal is allowed to perform write actions.
// Customer and partner users may never write, and vendor users require a role other than read-only.
func authorizeWrite(auth authinfo.AuthInfoWithOrganization) error {
	if auth.CurrentCustomerOrgID() != nil {
		return NewErrAccessDenied("customer user can not perform write action")
	}

	if auth.CurrentPartnerOrgID() != nil {
		return NewErrAccessDenied("partner user can not perform write action")
	}

	if auth.CurrentUserRole() == nil {
		return NewErrAccessDenied("user with no role can not perform write action")
	}

	if *auth.CurrentUserRole() == types.UserRoleReadOnly {
		return NewErrAccessDenied("read-only user can not perform write action")
	}

	return nil
}

// authorizeAnonymous is the only thing a request without credentials may do: read an artifact that
// its organization has made public.
func authorizeAnonymous(ctx context.Context, n *name.Name, action Action) error {
	if action != ActionRead && action != ActionStat {
		return ErrAuthenticationRequired
	} else if artifact, err := db.GetArtifactByName(ctx, n.OrgName, n.ArtifactName); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return ErrAuthenticationRequired
		}
		return err
	} else if !artifact.Public {
		return ErrAuthenticationRequired
	} else if AnonymousLimitExceeded(ctx) {
		return ErrRateLimited
	}
	return nil
}

func authorizeAnonymousBlob(ctx context.Context, n *name.Name, digest digest.Digest, action Action) error {
	// The blob route carries the repository name, so an anonymous caller is held to the artifact
	// it names rather than to any artifact of the organization that references the digest.
	if action != ActionRead && action != ActionStat {
		return ErrAuthenticationRequired
	} else if belongs, err := db.ArtifactBlobBelongsToPublicArtifact(
		ctx, n.OrgName, n.ArtifactName, digest.String(),
	); err != nil {
		return err
	} else if !belongs {
		return ErrAuthenticationRequired
	} else if AnonymousLimitExceeded(ctx) {
		return ErrRateLimited
	}
	return nil
}

// authorizeOrganization reports whether the principal's organization owns the reference. A principal
// reading another organization's repository is held to authorizePublic, the check an anonymous caller
// gets, because a host logged in to a registry domain sends its credentials for every repository on it.
func authorizeOrganization(org *types.Organization, n *name.Name, authorizePublic func() error) (bool, error) {
	if org.Slug != nil && *org.Slug == n.OrgName {
		return true, nil
	} else if err := authorizePublic(); !errors.Is(err, ErrAuthenticationRequired) {
		return false, err
	} else if org.Slug == nil {
		return false, NewErrAccessDenied("organization has no slug")
	} else {
		return false, NewErrAccessDenied("organization slug does not match reference")
	}
}

// authorizeUpstreamWrite rejects a push to a pull-through cache artifact.
func authorizeUpstreamWrite(ctx context.Context, n *name.Name) error {
	if artifact, err := db.GetArtifactByName(ctx, n.OrgName, n.ArtifactName); err != nil {
		if !errors.Is(err, distrerr.ErrNotFound) {
			return err
		}
	} else if artifact.UpstreamURL != nil {
		return NewErrAccessDenied("cannot push to a pull-through cache artifact")
	}
	return nil
}

// Authorize implements ArtifactsAuthorizer.
func (a *authorizer) Authorize(ctx context.Context, nameStr string, action Action) error {
	n, err := name.Parse(nameStr)
	if err != nil {
		return err
	}

	principal, ok := auth.ArtifactsPrincipal(ctx)
	if !ok {
		return authorizeAnonymous(ctx, n, action)
	}

	if action == ActionWrite {
		if err := authorizeWrite(principal); err != nil {
			return err
		}
	}

	org := principal.CurrentOrg()
	if owned, err := authorizeOrganization(org, n, func() error {
		return authorizeAnonymous(ctx, n, action)
	}); !owned || err != nil {
		return err
	}

	if action == ActionWrite {
		return authorizeUpstreamWrite(ctx, n)
	}

	return nil
}

// AuthorizeReference implements ArtifactsAuthorizer.
func (a *authorizer) AuthorizeReference(ctx context.Context, nameStr string, reference string, action Action) error {
	n, err := name.Parse(nameStr)
	if err != nil {
		return err
	}

	principal, ok := auth.ArtifactsPrincipal(ctx)
	if !ok {
		return authorizeAnonymous(ctx, n, action)
	}

	if action == ActionWrite {
		if err := authorizeWrite(principal); err != nil {
			return err
		}
	}

	org := principal.CurrentOrg()
	if owned, err := authorizeOrganization(org, n, func() error {
		return authorizeAnonymous(ctx, n, action)
	}); !owned || err != nil {
		return err
	}

	if action == ActionWrite {
		return authorizeUpstreamWrite(ctx, n)
	}

	if principal.CurrentCustomerOrgID() != nil && org.HasFeature(types.FeatureLicensing) {
		err := db.CheckEntitlementForArtifact(ctx,
			n.OrgName,
			n.ArtifactName,
			reference,
			*principal.CurrentCustomerOrgID(),
			*principal.CurrentOrgID(),
		)
		if errors.Is(err, distrerr.ErrForbidden) {
			return NewErrAccessDenied("entitlement required")
		} else if err != nil {
			return err
		}
	}

	return nil
}

// AuthorizeBlob implements ArtifactsAuthorizer.
func (a *authorizer) AuthorizeBlob(
	ctx context.Context,
	nameStr string,
	digest digest.Digest,
	action Action,
) error {
	n, err := name.Parse(nameStr)
	if err != nil {
		return err
	}

	principal, ok := auth.ArtifactsPrincipal(ctx)
	if !ok {
		return authorizeAnonymousBlob(ctx, n, digest, action)
	}

	// For writes we skip ownership checks: the push may (re)associate this digest with the org, even if it already exists.
	if action == ActionWrite {
		return authorizeWrite(principal)
	}

	org := principal.CurrentOrg()
	if owned, err := authorizeOrganization(org, n, func() error {
		return authorizeAnonymousBlob(ctx, n, digest, action)
	}); !owned || err != nil {
		return err
	}

	// A pushing client stats every blob after uploading it and before any manifest references it.
	if action != ActionStat {
		if belongs, err := db.ArtifactBlobBelongsToOrg(ctx, org.ID, digest.String()); err != nil {
			return err
		} else if !belongs {
			return distrerr.ErrNotFound
		}
	}

	if principal.CurrentCustomerOrgID() != nil && org.HasFeature(types.FeatureLicensing) {
		err := db.CheckEntitlementForArtifactBlob(
			ctx, digest.String(), n.ArtifactName, *principal.CurrentCustomerOrgID(), *principal.CurrentOrgID())
		if errors.Is(err, distrerr.ErrForbidden) {
			return NewErrAccessDenied("entitlement required")
		} else if err != nil {
			return err
		}
	}

	return nil
}
