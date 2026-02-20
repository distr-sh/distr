package apierrors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrConflict      = errors.New("conflict")
	ErrBadRequest    = errors.New("bad request")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrQuotaExceeded = errors.New("quota exceeded")
)

// NewBadRequest creates a new bad request error with the given message
func NewBadRequest(message string) error {
	return fmt.Errorf("%w: %s", ErrBadRequest, message)
}

// NewConflict creates a new conflict error with the given message
func NewConflict(message string) error {
	return fmt.Errorf("%w: %s", ErrConflict, message)
}
