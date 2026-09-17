package authz

import (
	"errors"
	"fmt"
)

var ErrAccessDenied = errors.New("access denied")

// ErrAuthenticationRequired is every refusal of an anonymous request, whether the artifact is
// private, missing or the action is a write. OCI clients authenticate only after a 401 carrying the
// challenge, so a denial or a not-found would stop a client that holds credentials from ever
// retrying with them, and a denial would additionally tell an anonymous caller that a private
// artifact exists.
var ErrAuthenticationRequired = errors.New("authentication required")

func NewErrAccessDenied(message string) error {
	return fmt.Errorf("%w: %s", ErrAccessDenied, message)
}
