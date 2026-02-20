package authz

import (
	"errors"
	"fmt"
)

var ErrAccessDenied = errors.New("access denied")

func NewErrAccessDenied(message string) error {
	return fmt.Errorf("%w: %s", ErrAccessDenied, message)
}
