package api

import (
	"strings"
	"time"

	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type CreateUpdatePartnerOrganizationRequest struct {
	Name string `json:"name"`
}

func (r *CreateUpdatePartnerOrganizationRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return validation.NewValidationFailedError("name is required")
	}
	return nil
}

type PartnerOrganization struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Name      string    `json:"name"`
}

type PartnerOrganizationWithUsage struct {
	PartnerOrganization
	UserCount                 int64 `json:"userCount"`
	CustomerOrganizationCount int64 `json:"customerOrganizationCount"`
}

type AssignCustomerToPartnerRequest struct {
	PartnerOrganizationID *uuid.UUID `json:"partnerOrganizationId"`
}
