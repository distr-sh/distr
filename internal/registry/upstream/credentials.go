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
	"github.com/google/uuid"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type cachedECRToken struct {
	username  string
	password  string
	expiresAt time.Time
}

var (
	ecrTokenCache   = map[uuid.UUID]cachedECRToken{}
	ecrTokenCacheMu sync.Mutex
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
	ecrTokenCacheMu.Lock()
	defer ecrTokenCacheMu.Unlock()

	if cached, ok := ecrTokenCache[artifact.ID]; ok && time.Now().Before(cached.expiresAt) {
		return auth.Credential{Username: cached.username, Password: cached.password}, nil
	}

	if artifact.UpstreamURL == nil {
		return auth.Credential{}, fmt.Errorf("ECR artifact %s has no upstream URL", artifact.ID)
	}
	region := ecrRegionFromURL(*artifact.UpstreamURL)
	if region == "" {
		return auth.Credential{}, fmt.Errorf("could not determine AWS region from upstream URL for artifact %s", artifact.ID)
	}
	if artifact.UpstreamUsername == nil || artifact.UpstreamPassword == nil {
		return auth.Credential{}, fmt.Errorf("missing AWS credentials for ECR artifact %s", artifact.ID)
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
		return auth.Credential{}, fmt.Errorf("loading AWS config: %w", err)
	}

	result, err := ecr.NewFromConfig(cfg).GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return auth.Credential{}, fmt.Errorf("getting ECR authorization token: %w", err)
	}
	if len(result.AuthorizationData) == 0 {
		return auth.Credential{}, fmt.Errorf("no ECR authorization data returned")
	}

	decoded, err := base64.StdEncoding.DecodeString(*result.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return auth.Credential{}, fmt.Errorf("decoding ECR token: %w", err)
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return auth.Credential{}, fmt.Errorf("unexpected ECR token format")
	}

	expiresAt := time.Now().Add(11 * time.Hour)
	if result.AuthorizationData[0].ExpiresAt != nil {
		expiresAt = result.AuthorizationData[0].ExpiresAt.Add(-1 * time.Hour)
	}
	ecrTokenCache[artifact.ID] = cachedECRToken{
		username:  username,
		password:  password,
		expiresAt: expiresAt,
	}

	return auth.Credential{Username: username, Password: password}, nil
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
