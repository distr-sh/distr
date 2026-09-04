package dbcrypto

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

// ErrNotEncrypted is returned when a value of an encrypted column is passed to a query as itself.
// Such a value has to be sealed with Encrypt and written to the encrypted column instead, and
// failing here is what keeps a plaintext write from succeeding unnoticed.
var ErrNotEncrypted = errors.New("value must be sealed with Encrypt before it is written")

// String is the value of an encrypted TEXT column. Scanning it decrypts what [TextColumn] read.
type String string

func (s *String) Scan(src any) error {
	value, err := srcBytes(src)
	if err != nil {
		return err
	}
	if value == nil {
		*s = ""
		return nil
	}
	plaintext, err := Keys().Decrypt(value)
	if err != nil {
		return err
	}
	*s = String(plaintext)
	return nil
}

func (String) Value() (driver.Value, error) { return nil, ErrNotEncrypted }

func (s String) Encrypt() ([]byte, error) { return Keys().Encrypt([]byte(s)) }

// Bytes is the value of an encrypted BYTEA column. Scanning it decrypts what [BytesColumn] read.
type Bytes []byte

func (b *Bytes) Scan(src any) error {
	value, err := srcBytes(src)
	if err != nil {
		return err
	}
	if value == nil {
		*b = nil
		return nil
	}
	plaintext, err := Keys().Decrypt(value)
	if err != nil {
		return err
	}
	*b = plaintext
	return nil
}

func (Bytes) Value() (driver.Value, error) { return nil, ErrNotEncrypted }

// Encrypt keeps nil as NULL, because a nullable column of this kind uses NULL to mean that there is
// no value rather than that there is an empty one.
func (b Bytes) Encrypt() ([]byte, error) {
	if b == nil {
		return nil, nil
	}
	return Keys().Encrypt(b)
}

// EncryptString seals a nullable [String], keeping nil as NULL.
func EncryptString(s *String) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	return s.Encrypt()
}

// StringPtr converts the nullable plain string of an api type into a nullable [String].
func StringPtr(s *string) *String {
	if s == nil {
		return nil
	}
	return new(String(*s))
}

func srcBytes(src any) ([]byte, error) {
	switch value := src.(type) {
	case nil:
		return nil, nil
	case []byte:
		return value, nil
	case string:
		return []byte(value), nil
	default:
		return nil, fmt.Errorf("cannot scan %T into an encrypted value", src)
	}
}
