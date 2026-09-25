package kms

import (
	"context"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awskms "github.com/aws/aws-sdk-go-v2/service/kms"
)

type AWSConfig struct {
	KeyID    string
	Region   *string
	Endpoint *string
}

// region falls back to the one a key ARN names, and otherwise to the SDK's own resolution.
func (c AWSConfig) region() string {
	if c.Region != nil {
		return *c.Region
	}
	// arn:aws:kms:<region>:<account>:key/<id>
	if fields := strings.Split(c.KeyID, ":"); len(fields) > 3 && fields[0] == "arn" {
		return fields[3]
	}
	return ""
}

type awsProvider struct {
	client *awskms.Client
	keyID  string
}

func newAWSProvider(ctx context.Context, cfg Config) (provider, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not load AWS configuration: %w", err)
	}
	client := awskms.NewFromConfig(awsCfg, func(o *awskms.Options) {
		if region := cfg.AWS.region(); region != "" {
			o.Region = region
		}
		o.BaseEndpoint = cfg.AWS.Endpoint
	})
	return &awsProvider{client: client, keyID: cfg.AWS.KeyID}, nil
}

// Decrypt names the key although the ciphertext already records it, as AWS recommends for a
// symmetric key: the call then fails on a ciphertext of another key instead of opening it.
func (p *awsProvider) Decrypt(ctx context.Context, ciphertext []byte, setting string) ([]byte, error) {
	out, err := p.client.Decrypt(ctx, &awskms.DecryptInput{
		KeyId:             &p.keyID,
		CiphertextBlob:    ciphertext,
		EncryptionContext: encryptionContext(setting),
	})
	if err != nil {
		return nil, err
	}
	return out.Plaintext, nil
}

func (p *awsProvider) Close() error { return nil }
