package audit

import (
	"context"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type ArtifactAuditor interface {
	AuditPull(ctx context.Context, name, reference string) error
}

type auditor struct{}

func NewAuditor() ArtifactAuditor {
	return &auditor{}
}

// AuditPull implements ArtifactAuditor.
func (a *auditor) AuditPull(ctx context.Context, nameStr string, reference string) error {
	name, err := name.Parse(nameStr)
	if err != nil {
		return err
	}
	digestVersion, err := db.GetArtifactVersion(ctx, name.OrgName, name.ArtifactName, reference)
	if err != nil {
		return err
	}
	// A principal pulling another organization's public artifact is logged as anonymous, so the
	// artifact owner never sees the identity or deployment targets of a foreign organization.
	if principal, ok := auth.ArtifactsPrincipal(ctx); ok && ownsArtifact(principal.CurrentOrg(), name) {
		return db.CreateArtifactPullLogEntry(
			ctx,
			digestVersion.ID,
			principal.CurrentUserID(),
			chimiddleware.GetClientIP(ctx),
			principal.CurrentCustomerOrgID(),
			principal.CurrentDeploymentTargetID(),
			false,
		)
	}
	return db.CreateArtifactPullLogEntry(
		ctx,
		digestVersion.ID,
		uuid.Nil,
		chimiddleware.GetClientIP(ctx),
		nil,
		nil,
		true,
	)
}

func ownsArtifact(org *types.Organization, n *name.Name) bool {
	return org != nil && org.Slug != nil && *org.Slug == n.OrgName
}
