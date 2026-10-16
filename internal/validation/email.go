package validation

import (
	"regexp"
)

var emailFormatPattern = regexp.MustCompile(`^[^\s@.]+(\.[^\s@.]+)*@[^\s@.]+(\.[^\s@.]+)+$`)

func ValidateEmail(email string) error {
	if !emailFormatPattern.MatchString(email) {
		return NewValidationFailedError("invalid email format")
	}
	return nil
}
