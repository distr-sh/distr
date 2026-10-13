package apierrors

import (
	"errors"
	"net/http"
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

var sentinelStatuses = []struct {
	err    error
	status int
}{
	{ErrNotFound, http.StatusNotFound},
	{ErrAlreadyExists, http.StatusConflict},
	{ErrConflict, http.StatusConflict},
	{ErrBadRequest, http.StatusBadRequest},
	{ErrUnauthorized, http.StatusUnauthorized},
	{ErrForbidden, http.StatusForbidden},
	{ErrQuotaExceeded, http.StatusForbidden},
}

// Error carries a Message sent to the end user verbatim. [errors.Is] matches its sentinel.
type Error struct {
	Status   int
	Message  string
	sentinel error
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Is(target error) bool {
	return e.sentinel != nil && target == e.sentinel
}

// New is for a status without a sentinel.
func New(status int, message string) error {
	return &Error{Status: status, Message: message}
}

func NewBadRequest(message string) error {
	return &Error{Status: http.StatusBadRequest, Message: message, sentinel: ErrBadRequest}
}

func NewNotFound(message string) error {
	return &Error{Status: http.StatusNotFound, Message: message, sentinel: ErrNotFound}
}

func NewConflict(message string) error {
	return &Error{Status: http.StatusConflict, Message: message, sentinel: ErrConflict}
}

func NewForbidden(message string) error {
	return &Error{Status: http.StatusForbidden, Message: message, sentinel: ErrForbidden}
}

// SentinelStatus returns the status of the first sentinel err wraps. Never send err.Error() along,
// wrapped sentinels often carry internal details.
func SentinelStatus(err error) (int, bool) {
	for _, s := range sentinelStatuses {
		if errors.Is(err, s.err) {
			return s.status, true
		}
	}
	return 0, false
}
