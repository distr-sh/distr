package db

import (
	"github.com/distr-sh/distr/internal/types"
	"github.com/jackc/pgx/v5"
)

// activityUserExpr selects the column of the user u, whose membership in the organization is j, when the viewer
// passed with activityViewerArgs may learn who the user is. customerOrgID is the SQL expression for the customer
// organization the activity belongs to.
func activityUserExpr(column, customerOrgID string) string {
	return `CASE WHEN j.customer_organization_id = ` + customerOrgID + `
			OR j.partner_organization_id = @viewerPartnerOrgId
			OR (@viewerIsVendor AND j.user_account_id IS NOT NULL)
		THEN u.` + column + ` END`
}

func activityViewerArgs(viewer types.ActivityViewer, args pgx.NamedArgs) pgx.NamedArgs {
	args["viewerIsVendor"] = viewer.IsVendor
	args["viewerPartnerOrgId"] = viewer.PartnerOrganizationID
	return args
}
