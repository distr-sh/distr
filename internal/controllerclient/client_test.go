package controllerclient

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/distr-sh/distr/internal/httpstatus"
	. "github.com/onsi/gomega"
)

func TestRejected(t *testing.T) {
	g := NewWithT(t)
	status := func(code int) error {
		return fmt.Errorf("export: %w", &httpstatus.StatusError{StatusCode: code})
	}

	for _, code := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusRequestEntityTooLarge} {
		g.Expect(rejected(status(code))).To(BeTrue(), "status %d", code)
	}
	for _, code := range []int{
		http.StatusRequestTimeout,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
	} {
		g.Expect(rejected(status(code))).To(BeFalse(), "status %d", code)
	}
	g.Expect(rejected(errors.New("connection reset by peer"))).To(BeFalse())
}
