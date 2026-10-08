package security

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/distr-sh/distr/internal/types"
	"golang.org/x/crypto/argon2"
)

const (
	time    = 1
	memory  = 64 * 1024
	threads = 4
	keyLen  = 32
)

var (
	ErrInvalidPassword  = errors.New("invalid password")
	ErrInvalidAccessKey = errors.New("invalid accessKey")
	ErrNoAccessKey      = errors.New("no accessKey")
)

func HashPassword(userAccount *types.UserAccount) error {
	if salt, err := generateSalt(); err != nil {
		return err
	} else {
		userAccount.PasswordSalt = salt
		userAccount.PasswordHash = generateHash(userAccount.Password, salt)
		userAccount.Password = ""
		return nil
	}
}

func HashAccessKey(accessKey string) ([]byte, []byte, error) {
	if salt, err := generateSalt(); err != nil {
		return nil, nil, err
	} else {
		hash := generateHash(accessKey, salt)
		return salt, hash, nil
	}
}

func VerifyPassword(userAccount types.UserAccount, password string) error {
	if !bytes.Equal(userAccount.PasswordHash, generateHash(password, userAccount.PasswordSalt)) {
		return ErrInvalidPassword
	} else {
		return nil
	}
}

func VerifyAccessKey(accessKeySalt []byte, accessKeyHash []byte, accessKeySecret string) error {
	if !bytes.Equal(accessKeyHash, generateHash(accessKeySecret, accessKeySalt)) {
		return ErrInvalidAccessKey
	} else {
		return nil
	}
}

// VerifyDeploymentTargetAccessKey accepts the active and the pending secret of a deployment target and reports
// whether the pending one matched.
func VerifyDeploymentTargetAccessKey(dt types.DeploymentTarget, accessKeySecret string) (pending bool, err error) {
	hasActive := dt.AccessKeySalt != nil && dt.AccessKeyHash != nil
	hasPending := dt.PendingAccessKeySalt != nil && dt.PendingAccessKeyHash != nil
	if !hasActive && !hasPending {
		return false, ErrNoAccessKey
	}
	if hasActive && VerifyAccessKey(*dt.AccessKeySalt, *dt.AccessKeyHash, accessKeySecret) == nil {
		return false, nil
	}
	if hasPending && VerifyAccessKey(*dt.PendingAccessKeySalt, *dt.PendingAccessKeyHash, accessKeySecret) == nil {
		return true, nil
	}
	return false, ErrInvalidAccessKey
}

func generateHash(password string, salt []byte) []byte {
	return argon2.IDKey([]byte(password), salt, time, memory, threads, keyLen)
}

func generateSalt() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	return salt, err
}

func GenerateAccessKey() (string, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", err
	} else {
		return hex.EncodeToString(key), nil
	}
}
