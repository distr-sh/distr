package httpstatus

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ErrHttpStatus is the sentinel that every StatusError unwraps to, so callers can match
// any non-2xx response with errors.Is(err, ErrHttpStatus).
var ErrHttpStatus = errors.New("non-ok http status")

// StatusError describes a non-2xx HTTP response. Callers can use errors.As to extract the
// status code and react to specific statuses (e.g. treating 400 as a permanent failure).
type StatusError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *StatusError) Error() string {
	if e.Body != "" {
		return fmt.Sprintf("%v: %v (%v)", ErrHttpStatus, e.Status, e.Body)
	}
	return fmt.Sprintf("%v: %v", ErrHttpStatus, e.Status)
}

func (e *StatusError) Unwrap() error { return ErrHttpStatus }

func CheckStatus(r *http.Response, err error) (*http.Response, error) {
	if err != nil || StatusOK(r) {
		return r, err
	}
	// The response is only used to build the error, so drain and close the body here to
	// avoid leaking the connection/file descriptor since callers discard r on error.
	defer func() { _ = r.Body.Close() }()
	statusErr := &StatusError{StatusCode: r.StatusCode, Status: r.Status}
	if body, err := io.ReadAll(r.Body); err == nil {
		statusErr.Body = strings.TrimSpace(string(body))
	}
	return r, statusErr
}

func StatusOK(r *http.Response) bool {
	return 200 <= r.StatusCode && r.StatusCode < 300
}
