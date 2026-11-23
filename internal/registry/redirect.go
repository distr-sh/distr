package registry

import (
	"context"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
)

// redirectAllowed reports whether a pull from repo may be redirected to object storage, which the organization owning
// repo can override.
func redirectAllowed(ctx context.Context, repo string) (bool, error) {
	if n, err := name.Parse(repo); err != nil {
		return false, err
	} else if org, err := repoOrganization(ctx, n); err != nil {
		return false, err
	} else {
		return util.PtrDerefOr(org.RegistryRedirectEnabled, env.RegistryS3Config().AllowRedirect), nil
	}
}

func repoOrganization(ctx context.Context, n *name.Name) (*types.Organization, error) {
	if principal, ok := auth.ArtifactsPrincipal(ctx); ok {
		if org := principal.CurrentOrg(); org != nil && org.Slug != nil && *org.Slug == n.OrgName {
			return org, nil
		}
	}
	return db.GetOrganizationBySlug(ctx, n.OrgName)
}
