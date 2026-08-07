package handlers

import (
	"testing"

	"github.com/distr-sh/distr/internal/types"
	. "github.com/onsi/gomega"
)

func TestExpectedCNAMETarget(t *testing.T) {
	appTarget := "custom-app.distr.sh"
	registryTarget := "custom-registry.distr.sh"

	t.Run("app domain always uses the app target", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(expectedCNAMETarget(types.DomainTypeApp, &appTarget, &registryTarget)).To(HaveValue(Equal(appTarget)))
	})

	t.Run("customer portal domain uses the app target", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(expectedCNAMETarget(types.DomainTypeCustomerPortal, &appTarget, &registryTarget)).
			To(HaveValue(Equal(appTarget)))
	})

	t.Run("registry domain uses its own target when configured", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(expectedCNAMETarget(types.DomainTypeRegistry, &appTarget, &registryTarget)).
			To(HaveValue(Equal(registryTarget)))
	})

	t.Run("registry domain falls back to the app target when no registry target is configured", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(expectedCNAMETarget(types.DomainTypeRegistry, &appTarget, nil)).To(HaveValue(Equal(appTarget)))
	})

	t.Run("nil when nothing is configured", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(expectedCNAMETarget(types.DomainTypeApp, nil, nil)).To(BeNil())
	})
}
