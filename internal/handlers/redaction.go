package handlers

import (
	"context"
	"fmt"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/redact"
	"github.com/distr-sh/distr/internal/types"
)

// deploymentTargetRedactor masks the secrets a deployment target can reference as well as common
// credential patterns in what the target's controller reports.
func deploymentTargetRedactor(ctx context.Context, dt types.DeploymentTarget) (*redact.Redactor, error) {
	if secrets, err := db.GetSecretsForDeploymentTarget(ctx, dt); err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	} else {
		return secretsRedactor(secrets), nil
	}
}

func secretsRedactor(secrets []types.SecretWithUpdatedBy) *redact.Redactor {
	values := make([]string, 0, 2*len(secrets))
	for _, secret := range secrets {
		value := string(secret.Value)
		values = append(values, value)
		if escaped := deploymentvalues.EscapeNewlines(value); escaped != value {
			values = append(values, escaped)
		}
	}
	return redact.New(values...)
}

// queryTimeRedactor redacts log records that were stored before redaction moved to ingestion.
//
// TODO: Remove once no log records ingested without redaction remain within the default Loki
// retention period of 30 days.
func queryTimeRedactor(ctx context.Context, dt types.DeploymentTarget) (*redact.Redactor, error) {
	return deploymentTargetRedactor(ctx, dt)
}
