package mapping

import (
	"testing"
	"time"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	. "github.com/onsi/gomega"
)

func TestEnvironmentToAPI(t *testing.T) {
	g := NewWithT(t)
	id := uuid.New()
	orgID := uuid.New()
	retentionPolicyID := uuid.New()
	createdAt := time.Date(2026, 6, 20, 9, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 20, 10, 45, 0, 0, time.UTC)

	res := EnvironmentToAPI(types.Environment{
		ID:                  id,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
		OrganizationID:      orgID,
		Name:                "Production",
		Description:         "Customer production targets",
		SortOrder:           30,
		IsProduction:        true,
		AllowDynamicTargets: false,
		RetentionPolicyID:   &retentionPolicyID,
	})

	g.Expect(res).To(Equal(api.Environment{
		ID:                  id,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
		Name:                "Production",
		Description:         "Customer production targets",
		SortOrder:           30,
		IsProduction:        true,
		AllowDynamicTargets: false,
		RetentionPolicyID:   &retentionPolicyID,
	}))
}
