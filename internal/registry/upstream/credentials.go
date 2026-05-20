package upstream

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/distr-sh/distr/internal/types"
	"golang.org/x/sync/singleflight"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type cachedECRToken struct {
	username  string
	password  string
	expiresAt time.Time
}

var (
	ecrTokenCache   sync.Map // map[uuid.UUID]cachedECRToken
	ecrSingleflight singleflight.Group
)

func credentialForArtifact(artifact *types.Artifact) auth.CredentialFunc {
	if artifact.UpstreamAuthType == nil {
		return func(_ context.Context, _ string) (auth.Credential, error) { return auth.EmptyCredential, nil }
	}
	switch *artifact.UpstreamAuthType {
	case types.UpstreamAuthTypeBasic:
		if artifact.UpstreamUsername == nil || artifact.UpstreamPassword == nil {
			return func(_ context.Context, _ string) (auth.Credential, error) {
				return auth.Credential{}, fmt.Errorf("missing upstream credentials for artifact %s", artifact.ID)
			}
		}
		username := *artifact.UpstreamUsername
		password := *artifact.UpstreamPassword
		return func(_ context.Context, _ string) (auth.Credential, error) {
			return auth.Credential{Username: username, Password: password}, nil
		}
	case types.UpstreamAuthTypeAWSECR:
		return func(ctx context.Context, _ string) (auth.Credential, error) {
			return getECRCredential(ctx, artifact)
		}
	default:
		return func(_ context.Context, _ string) (auth.Credential, error) { return auth.EmptyCredential, nil }
	}
}

func getECRCredential(ctx context.Context, artifact *types.Artifact) (auth.Credential, error) {
	if v, ok := ecrTokenCache.Load(artifact.ID); ok {
		if cached := v.(cachedECRToken); time.Now().Before(cached.expiresAt) {
			return auth.Credential{Username: cached.username, Password: cached.password}, nil
		}
	}

	v, err, _ := ecrSingleflight.Do(artifact.ID.String(), func() (any, error) {
		// Double-check cache: another goroutine may have fetched the token while we waited.
		if v, ok := ecrTokenCache.Load(artifact.ID); ok {
			if cached := v.(cachedECRToken); time.Now().Before(cached.expiresAt) {
				return auth.Credential{Username: cached.username, Password: cached.password}, nil
			}
		}

		if artifact.UpstreamURL == nil {
			return nil, fmt.Errorf("ECR artifact %s has no upstream URL", artifact.ID)
		}
		region := ecrRegionFromURL(*artifact.UpstreamURL)
		if region == "" {
			return nil, fmt.Errorf("could not determine AWS region from upstream URL for artifact %s", artifact.ID)
		}
		if artifact.UpstreamUsername == nil || artifact.UpstreamPassword == nil {
			return nil, fmt.Errorf("missing AWS credentials for ECR artifact %s", artifact.ID)
		}

		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				*artifact.UpstreamUsername,
				*artifact.UpstreamPassword,
				"",
			)),
		)
		if err != nil {
			return nil, fmt.Errorf("loading AWS config: %w", err)
		}

		result, err := ecr.NewFromConfig(cfg).GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
		if err != nil {
			return nil, fmt.Errorf("getting ECR authorization token: %w", err)
		}
		if len(result.AuthorizationData) == 0 {
			return nil, fmt.Errorf("no ECR authorization data returned")
		}

		decoded, err := base64.StdEncoding.DecodeString(*result.AuthorizationData[0].AuthorizationToken)
		if err != nil {
			return nil, fmt.Errorf("decoding ECR token: %w", err)
		}
		username, password, ok := strings.Cut(string(decoded), ":")
		if !ok {
			return nil, fmt.Errorf("unexpected ECR token format")
		}

		expiresAt := time.Now().Add(11 * time.Hour)
		if result.AuthorizationData[0].ExpiresAt != nil {
			expiresAt = result.AuthorizationData[0].ExpiresAt.Add(-1 * time.Hour)
		}
		ecrTokenCache.Store(artifact.ID, cachedECRToken{username: username, password: password, expiresAt: expiresAt})

		return auth.Credential{Username: username, Password: password}, nil
	})
	if err != nil {
		return auth.Credential{}, err
	}
	return v.(auth.Credential), nil
}

// ValidateUpstreamCredentials checks that the configured credentials can authenticate
// against the upstream registry. It is a no-op when no auth type is set.
func ValidateUpstreamCredentials(ctx context.Context, artifact *types.Artifact) error {
	if artifact.UpstreamURL == nil || artifact.UpstreamAuthType == nil {
		return nil
	}
	if artifact.UpstreamUsername == nil || *artifact.UpstreamUsername == "" {
		return fmt.Errorf("username is required when upstream authentication is configured")
	}
	if artifact.UpstreamPassword == nil || *artifact.UpstreamPassword == "" {
		return fmt.Errorf("password is required when upstream authentication is configured")
	}
	repo, err := remote.NewRepository(*artifact.UpstreamURL)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	repo.Client = &auth.Client{Credential: credentialForArtifact(artifact)}

	reg, err := remote.NewRegistry(repo.Reference.Registry)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}
	reg.Client = repo.Client

	if err := reg.Ping(ctx); err != nil {
		return fmt.Errorf("upstream registry authentication failed: %w", err)
	}
	return nil
}

// ecrRegionFromURL parses the AWS region from an ECR URL.
// ECR URLs have the form: <account>.dkr.ecr.<region>.amazonaws.com/...
func ecrRegionFromURL(upstreamURL string) string {
	host := upstreamURL
	if i := strings.Index(host, "/"); i != -1 {
		host = host[:i]
	}
	parts := strings.Split(host, ".")
	for i, part := range parts {
		if part == "ecr" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
