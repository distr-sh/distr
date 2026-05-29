package audit

import (
	"context"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/registry/name"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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
	auth := auth.ArtifactsAuthentication.Require(ctx)
	if name, err := name.Parse(nameStr); err != nil {
		return err
	} else if digestVersion, err := db.GetArtifactVersion(ctx, name.OrgName, name.ArtifactName, reference); err != nil {
		return err
	} else {
		return db.CreateArtifactPullLogEntry(
			ctx,
			digestVersion.ID,
			auth.CurrentUserID(),
			chimiddleware.GetClientIP(ctx),
			auth.CurrentCustomerOrgID(),
		)
	}
}
