package handlers

import (
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestEnvironmentFromCreateUpdateRequest(t *testing.T) {
	g := NewWithT(t)
	orgID := uuid.New()
	retentionPolicyID := uuid.New()

	environment := environmentFromCreateUpdateRequest(orgID, api.CreateUpdateEnvironmentRequest{
		Name:                "Production",
		Description:         "Customer production targets",
		SortOrder:           30,
		IsProduction:        true,
		AllowDynamicTargets: false,
		RetentionPolicyID:   &retentionPolicyID,
	})

	g.Expect(environment).To(Equal(types.Environment{
		OrganizationID:      orgID,
		Name:                "Production",
		Description:         "Customer production targets",
		SortOrder:           30,
		IsProduction:        true,
		AllowDynamicTargets: false,
		RetentionPolicyID:   &retentionPolicyID,
	}))
}

func TestEnvironmentResponses(t *testing.T) {
	g := NewWithT(t)
	id := uuid.New()

	responses := environmentResponses([]types.Environment{{ID: id, Name: "Development"}})

	g.Expect(responses).To(Equal([]api.Environment{{ID: id, Name: "Development"}}))
}
