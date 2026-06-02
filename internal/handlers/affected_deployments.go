package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func findAffectedDeploymentsBySecret(
	ctx context.Context,
	orgID uuid.UUID,
	secretKey string,
	newValue string,
	customerOrgID *uuid.UUID,
) ([]api.AffectedDeployment, error) {
	return findAffectedDeployments(ctx, orgID, customerOrgID, func(
		secrets []types.SecretWithUpdatedBy,
		licenseKeys []types.LicenseKey,
	) ([]types.SecretWithUpdatedBy, []types.LicenseKey) {
		patchedSecrets := slices.Clone(secrets)
		for i := range patchedSecrets {
			if patchedSecrets[i].Key == secretKey {
				patchedSecrets[i].Value = newValue
				break
			}
		}
		return patchedSecrets, licenseKeys
	})
}

func findAffectedDeploymentsByLicenseKey(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID uuid.UUID,
	updatedLicenseKey types.LicenseKey,
) ([]api.AffectedDeployment, error) {
	return findAffectedDeployments(ctx, orgID, &customerOrgID, func(
		secrets []types.SecretWithUpdatedBy,
		licenseKeys []types.LicenseKey,
	) ([]types.SecretWithUpdatedBy, []types.LicenseKey) {
		patchedLicenseKeys := slices.Clone(licenseKeys)
		for i := range patchedLicenseKeys {
			if patchedLicenseKeys[i].ID == updatedLicenseKey.ID {
				patchedLicenseKeys[i] = updatedLicenseKey
				break
			}
		}
		return secrets, patchedLicenseKeys
	})
}

func findAffectedDeployments(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID *uuid.UUID,
	patch func([]types.SecretWithUpdatedBy, []types.LicenseKey) ([]types.SecretWithUpdatedBy, []types.LicenseKey),
) ([]api.AffectedDeployment, error) {
	targets, err := db.GetDeploymentTargetsByScope(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	}

	affected := make([]api.AffectedDeployment, 0)
	for _, target := range targets {
		deployments, err := db.GetDeploymentsForDeploymentTarget(ctx, target.ID)
		if err != nil {
			return nil, err
		}
		secrets, err := db.GetSecretsForDeploymentTarget(ctx, target)
		if err != nil {
			return nil, err
		}
		licenseKeys, err := db.GetLicenseKeysForDeploymentTarget(ctx, target)
		if err != nil {
			return nil, err
		}

		patchedSecrets, patchedLicenseKeys := patch(secrets, licenseKeys)
		for _, deployment := range deployments {
			if len(deployment.ValuesHash) != sha256.Size {
				continue
			}
			newHash, err := deploymentvalues.RenderAndHash(&deployment, patchedSecrets, patchedLicenseKeys)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(newHash[:], deployment.ValuesHash) {
				affected = append(affected, api.AffectedDeployment{
					DeploymentTargetID:   target.ID,
					DeploymentTargetName: target.Name,
					DeploymentID:         deployment.ID,
				})
			}
		}
	}
	return affected, nil
}

func triggerAffectedDeployments(ctx context.Context, affected []api.AffectedDeployment) error {
	for _, affectedDeployment := range affected {
		target, err := db.GetDeploymentTarget(ctx, affectedDeployment.DeploymentTargetID, nil, nil)
		if err != nil {
			return err
		}
		deployments, err := db.GetDeploymentsForDeploymentTarget(ctx, affectedDeployment.DeploymentTargetID)
		if err != nil {
			return err
		}
		index := slices.IndexFunc(deployments, func(deployment types.DeploymentWithLatestRevision) bool {
			return deployment.ID == affectedDeployment.DeploymentID
		})
		if index < 0 {
			return fmt.Errorf("deployment %s not found for deployment target %s",
				affectedDeployment.DeploymentID, affectedDeployment.DeploymentTargetID)
		}

		deployment := deployments[index]
		request := deploymentRequestFromLatestRevision(deployment)
		secrets, err := db.GetSecretsForDeploymentTarget(ctx, target.DeploymentTarget)
		if err != nil {
			return err
		}
		licenseKeys, err := db.GetLicenseKeysForDeploymentTarget(ctx, target.DeploymentTarget)
		if err != nil {
			return err
		}
		if err := setDeploymentRequestValuesHash(&request, secrets, licenseKeys); err != nil {
			return err
		}
		if _, err := db.CreateDeploymentRevision(ctx, &request); err != nil {
			return err
		}
	}
	return nil
}

func deploymentRequestFromLatestRevision(deployment types.DeploymentWithLatestRevision) api.DeploymentRequest {
	return api.DeploymentRequest{
		DeploymentID:             &deployment.ID,
		DeploymentTargetID:       deployment.DeploymentTargetID,
		ApplicationVersionID:     deployment.ApplicationVersionID,
		ApplicationEntitlementID: deployment.ApplicationEntitlementID,
		ReleaseName:              deployment.ReleaseName,
		ValuesYaml:               deployment.ValuesYaml,
		DockerType:               deployment.DockerType,
		EnvFileData:              deployment.EnvFileData,
		ForceRestart:             deployment.ForceRestart,
		IgnoreRevisionSkew:       deployment.IgnoreRevisionSkew,
		HelmOptions:              apiHelmOptionsFromInternal(deployment.HelmOptions),
	}
}

func apiHelmOptionsFromInternal(options *types.HelmOptions) *api.HelmOptions {
	if options == nil {
		return nil
	}
	return &api.HelmOptions{
		Timeout:           options.Timeout,
		WaitStrategy:      options.WaitStrategy,
		RollbackOnFailure: options.RollbackOnFailure,
		CleanupOnFailure:  options.CleanupOnFailure,
		ForceConflicts:    options.ForceConflicts,
	}
}
