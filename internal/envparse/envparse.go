package envparse

import (
	"errors"
	"net/mail"
	"strconv"
	"strings"
	"time"
)

func PositiveDuration(value string) (time.Duration, error) {
	parsed, err := time.ParseDuration(value)
	if err == nil && parsed.Nanoseconds() <= 0 {
		err = errors.New("duration must be positive")
	}
	return parsed, err
}

func ByteSlice(s string) ([]byte, error) {
	return []byte(s), nil
}

// Host accepts a host with an optional port. A scheme is rejected because the URL scheme of such a
// variable is taken from DISTR_HOST, and one given here would silently win over it.
func Host(value string) (string, error) {
	if value == "" || strings.Contains(value, "/") {
		return "", errors.New("must be a host with an optional port, without scheme or path")
	}
	return value, nil
}

func MailAddress(s string) (mail.Address, error) {
	if parsed, err := mail.ParseAddress(s); err != nil || parsed == nil {
		return mail.Address{}, err
	} else {
		return *parsed, nil
	}
}

func NonNegativeNumber(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err == nil && parsed < 0 {
		err = errors.New("number must not be negative")
	}
	return parsed, err
}

func PositiveNumber(value string) (int, error) {
	parsed, err := strconv.Atoi(value)
	if err == nil && parsed <= 0 {
		err = errors.New("number must be positive")
	}
	return parsed, err
}

func Float(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}
