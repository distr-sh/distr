package security

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"strings"
)

const (
	recoveryCodeLength = 10
	recoveryCodeCount  = 10
)

func GenerateRecoveryCodes() ([]string, error) {
	codes := make([]string, recoveryCodeCount)
	for i := range recoveryCodeCount {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code
	}
	return codes, nil
}

func generateRecoveryCode() (string, error) {
	codeBytes := make([]byte, 6)
	if _, err := rand.Read(codeBytes); err != nil {
		return "", err
	}
	hexString := hex.EncodeToString(codeBytes)
	return hexString[:recoveryCodeLength], nil
}

func FormatRecoveryCode(code string) string {
	if len(code) != recoveryCodeLength {
		return code
	}
	return code[:5] + "-" + code[5:]
}

func NormalizeRecoveryCode(code string) string {
	return strings.ToLower(strings.ReplaceAll(code, "-", ""))
}

func HashRecoveryCode(code string) ([]byte, []byte, error) {
	salt, err := generateSalt()
	if err != nil {
		return nil, nil, err
	}
	normalized := NormalizeRecoveryCode(code)
	hash := generateHash(normalized, salt)
	return salt, hash, nil
}

func VerifyRecoveryCode(code string, salt []byte, hash []byte) bool {
	normalized := NormalizeRecoveryCode(code)
	return bytes.Equal(hash, generateHash(normalized, salt))
}
