package kms

import (
	"context"
	"fmt"

	cloudkms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
)

type GCPConfig struct {
	KeyName string
}

type gcpProvider struct {
	client  *cloudkms.KeyManagementClient
	keyName string
}

// newGCPProvider authenticates with Application Default Credentials, so a service account key file
// is passed through GOOGLE_APPLICATION_CREDENTIALS.
func newGCPProvider(ctx context.Context, cfg Config) (provider, error) {
	client, err := cloudkms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not create GCP Cloud KMS client: %w", err)
	}
	return &gcpProvider{client: client, keyName: cfg.GCP.KeyName}, nil
}

func (p *gcpProvider) Decrypt(ctx context.Context, ciphertext []byte, setting string) ([]byte, error) {
	resp, err := p.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:                        p.keyName,
		Ciphertext:                  ciphertext,
		AdditionalAuthenticatedData: additionalAuthenticatedData(setting),
	})
	if err != nil {
		return nil, err
	}
	return resp.GetPlaintext(), nil
}

func (p *gcpProvider) Close() error { return p.client.Close() }
