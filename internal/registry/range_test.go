package registry

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestParseRange(t *testing.T) {
	g := NewWithT(t)

	for _, tc := range []struct {
		header     string
		start, end int64
	}{
		{"bytes=0-99", 0, 99},
		{"bytes=10-19", 10, 19},
		{"bytes=90-", 90, 99},
		{"bytes=50-1000", 50, 99},
		{"bytes=-10", 90, 99},
		{"bytes=-1000", 0, 99},
	} {
		start, end, isRange, err := parseRange(tc.header, 100)
		g.Expect(err).NotTo(HaveOccurred(), tc.header)
		g.Expect(isRange).To(BeTrue(), tc.header)
		g.Expect([]int64{start, end}).To(Equal([]int64{tc.start, tc.end}), tc.header)
	}

	for _, header := range []string{
		"bytes=0-9,20-29",
		"items=0-9",
	} {
		_, _, isRange, err := parseRange(header, 100)
		g.Expect(err).NotTo(HaveOccurred(), header)
		g.Expect(isRange).To(BeFalse(), header)
	}

	for _, tc := range []struct {
		header string
		size   int64
	}{
		{"bytes=100-", 100},
		{"bytes=100-199", 100},
		{"bytes=20-10", 100},
		{"bytes=-0", 100},
		{"bytes=-", 100},
		{"bytes=10", 100},
		{"bytes=+10-20", 100},
		{"bytes=-10", 0},
	} {
		_, _, _, err := parseRange(tc.header, tc.size)
		g.Expect(err).To(HaveOccurred(), tc.header)
	}
}
