package types

import "github.com/google/uuid"

// ActivityUser is the user who did something the activity feed shows, with their membership in the vendor's
// organization. Deleted is set when that user is no longer a member.
type ActivityUser struct {
	Name                   *string    `db:"name"`
	ImageID                *uuid.UUID `db:"image_id"`
	CustomerOrganizationID *uuid.UUID `db:"customer_organization_id"`
	PartnerOrganizationID  *uuid.UUID `db:"partner_organization_id"`
	Deleted                bool       `db:"deleted"`
}
