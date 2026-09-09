// Package kms unwraps a configuration value that is stored as the ciphertext of an external key
// management service, so that key material never appears in the configuration of an instance.
package kms

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// Prefix marks a value as a base64 ciphertext rather than a literal. It cannot collide with one:
// a keyring entry is "<numeric id>:<base64 key>" and a base64 secret holds no colon at all.
const Prefix = "kms:"

// contextKey is part of what an operator passes to their cloud CLI to wrap a value, so changing it
// locks every instance out of the values it has already wrapped.
const contextKey = "distr"

type Config struct {
	AWS AWSConfig
	GCP GCPConfig
}

type registration struct {
	keyVariable string
	keyOf       func(Config) string
	newProvider func(context.Context, Config) (provider, error)
}

// providers is the registry of every key management service: a new one is a file next to this one
// plus an entry here. Naming a key selects its service, so no separate setting picks one.
var providers = []registration{
	{"KMS_AWS_KEY_ID", func(c Config) string { return c.AWS.KeyID }, newAWSProvider},
	{"KMS_GCP_KEY_NAME", func(c Config) string { return c.GCP.KeyName }, newGCPProvider},
}

func keyVariables() []string {
	variables := make([]string, 0, len(providers))
	for _, registered := range providers {
		variables = append(variables, registered.keyVariable)
	}
	return variables
}

// provider needs no counterpart to Decrypt: a ciphertext is produced once by an operator with the
// CLI of their cloud, never by this code.
type provider interface {
	// Decrypt opens a ciphertext bound to the given setting as its encryption context.
	Decrypt(ctx context.Context, ciphertext []byte, setting string) ([]byte, error)
	io.Closer
}

// Resolver unwraps values with the one service an instance is configured with. Its zero value
// resolves literals and rejects everything else.
type Resolver struct {
	provider provider
}

// New picks the service whose key cfg names. Naming several is an error rather than resolved by
// precedence, so a key left behind by a migration between two clouds cannot decide which service
// an instance talks to.
func New(ctx context.Context, cfg Config) (*Resolver, error) {
	var configured []registration
	for _, registered := range providers {
		if registered.keyOf(cfg) != "" {
			configured = append(configured, registered)
		}
	}
	if len(configured) == 0 {
		return &Resolver{}, nil
	}
	if len(configured) > 1 {
		named := make([]string, 0, len(configured))
		for _, registered := range configured {
			named = append(named, registered.keyVariable)
		}
		return nil, fmt.Errorf("only one key management service can be configured, but %v are set",
			strings.Join(named, " and "))
	}
	provider, err := configured[0].newProvider(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &Resolver{provider: provider}, nil
}

// Resolve returns value unchanged unless it carries [Prefix], in which case the remainder is its
// base64 ciphertext, bound to setting as the encryption context so that a ciphertext minted for one
// setting cannot be used for another. A wrapped value with no service configured is an error rather
// than a literal, since a ciphertext must never reach code that expects key material.
func (r *Resolver) Resolve(ctx context.Context, setting, value string) (string, error) {
	encoded, wrapped := strings.CutPrefix(value, Prefix)
	if !wrapped {
		return value, nil
	}
	if r.provider == nil {
		return "", fmt.Errorf("%v is wrapped with a key management service, but none is configured. Set one of %v",
			setting, strings.Join(keyVariables(), ", "))
	}
	ciphertext, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return "", fmt.Errorf("%v is not valid base64 behind %q: %w", setting, Prefix, err)
	}
	plaintext, err := r.provider.Decrypt(ctx, ciphertext, setting)
	if err != nil {
		return "", fmt.Errorf("could not unwrap %v: %w", setting, err)
	}
	return string(plaintext), nil
}

func (r *Resolver) Close() error {
	if r.provider == nil {
		return nil
	}
	return r.provider.Close()
}

// encryptionContext is what `aws kms encrypt --encryption-context distr=<setting>` binds.
func encryptionContext(setting string) map[string]string {
	return map[string]string{contextKey: setting}
}

// additionalAuthenticatedData is [encryptionContext] in the shape Cloud KMS takes it, which is what
// an operator passes to `gcloud kms encrypt --additional-authenticated-data-file`.
func additionalAuthenticatedData(setting string) []byte {
	return []byte(contextKey + "=" + setting)
}
