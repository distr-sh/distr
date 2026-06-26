package featureflags

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseEnabledKeys(t *testing.T) {
	g := NewWithT(t)

	keys, err := ParseEnabledKeys("release_bundles, environments\nlifecycles release_bundles")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(keys).To(Equal([]Key{KeyEnvironments, KeyLifecycles, KeyReleaseBundles}))
}

func TestParseEnabledKeysAll(t *testing.T) {
	g := NewWithT(t)

	keys, err := ParseEnabledKeys("all")

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(keys).To(Equal(AllKeys()))
}

func TestParseEnabledKeysRejectsUnknownFlags(t *testing.T) {
	g := NewWithT(t)

	_, err := ParseEnabledKeys("environments,not_a_flag")

	g.Expect(err).To(MatchError(ContainSubstring(`unknown experimental feature flag "not_a_flag"`)))
}

func TestRegistryMarksEnabledFlags(t *testing.T) {
	g := NewWithT(t)

	registry := NewRegistry([]Key{KeyReleaseBundles, KeyEnvironments})
	flags := registry.Flags()
	environments := findFlag(flags, KeyEnvironments)
	lifecycles := findFlag(flags, KeyLifecycles)

	g.Expect(registry.IsEnabled(KeyEnvironments)).To(BeTrue())
	g.Expect(registry.IsEnabled(KeyReleaseBundles)).To(BeTrue())
	g.Expect(registry.IsEnabled(KeyLifecycles)).To(BeFalse())
	g.Expect(environments.Key).To(Equal(KeyEnvironments))
	g.Expect(environments.Label).To(Equal("Environments"))
	g.Expect(environments.Description).NotTo(BeEmpty())
	g.Expect(environments.Milestone).To(Equal("Milestone B"))
	g.Expect(environments.Enabled).To(BeTrue())
	g.Expect(lifecycles.Key).To(Equal(KeyLifecycles))
	g.Expect(lifecycles.Enabled).To(BeFalse())
}

func findFlag(flags []Flag, key Key) Flag {
	for _, flag := range flags {
		if flag.Key == key {
			return flag
		}
	}
	return Flag{}
}
