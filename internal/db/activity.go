package db

// activityUserOutputExpr selects the user joined with activityUserJoinExpr as a types.ActivityUser, or NULL when
// there is none.
const activityUserOutputExpr = `
	CASE WHEN u.id IS NOT NULL THEN (
		u.name,
		u.image_id,
		j.customer_organization_id,
		j.partner_organization_id,
		j.user_account_id IS NULL
	) END`

// activityUserJoinExpr joins the user userIDExpr selects and their membership in the vendor's organization, which
// orgIDExpr has to select.
func activityUserJoinExpr(userIDExpr, orgIDExpr string) string {
	return `
		LEFT JOIN UserAccount u ON u.id = ` + userIDExpr + `
		LEFT JOIN Organization_UserAccount j ON j.user_account_id = u.id AND j.organization_id = ` + orgIDExpr
}
