package handlers

import (
	"testing"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/featureflags"
	. "github.com/onsi/gomega"
)

func TestExperimentalFeatureFlagResponses(t *testing.T) {
	g := NewWithT(t)

	responses := experimentalFeatureFlagResponses([]featureflags.Flag{
		{
			Key:         featureflags.KeyEnvironments,
			Label:       "Environments",
			Description: "Groups deployment targets by promotion stage or operational purpose.",
			Milestone:   "Milestone B",
			Enabled:     true,
		},
	})

	g.Expect(responses).To(Equal([]api.ExperimentalFeatureFlag{
		{
			Key:         "environments",
			Label:       "Environments",
			Description: "Groups deployment targets by promotion stage or operational purpose.",
			Milestone:   "Milestone B",
			Enabled:     true,
		},
	}))
}
