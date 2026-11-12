package types

import "github.com/google/uuid"

// ActivityViewer is who an activity feed is assembled for. A customer only learns the names of the members of
// its own organization, a partner also those of its own members and the vendor those of every member.
type ActivityViewer struct {
	IsVendor              bool
	PartnerOrganizationID *uuid.UUID
}
