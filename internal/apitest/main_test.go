package apitest_test

import (
	"os"
	"testing"

	"github.com/distr-sh/distr/internal/apitest"
)

func TestMain(m *testing.M) {
	apitest.Main(m)
}

// knownBug skips a test that fails because of the bug described by reason, unless
// APITEST_KNOWN_BUGS=1 asks to run it, which shows whether the bug is still there.
func knownBug(t *testing.T, reason string) {
	t.Helper()
	if os.Getenv("APITEST_KNOWN_BUGS") != "1" {
		t.Skip("known bug: " + reason)
	}
}
