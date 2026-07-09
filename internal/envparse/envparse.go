package envparse

import (
	"errors"
	"net/mail"
	"strconv"
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
