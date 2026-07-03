package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
)

func updateSecretValuePatchFunc(
	secretKey string, newValue string,
) func([]types.SecretWithUpdatedBy) []types.SecretWithUpdatedBy {
	return func(secrets []types.SecretWithUpdatedBy) []types.SecretWithUpdatedBy {
		patched := slices.Clone(secrets)
		for i := range patched {
			if patched[i].Key == secretKey {
				patched[i].Value = newValue
				break
			}
		}
		return patched
	}
}

func findAffectedDeploymentsBySecret(
	ctx context.Context,
	orgID uuid.UUID,
	secretKey string,
	newValue string,
	customerOrgID *uuid.UUID,
) ([]api.AffectedDeployment, error) {
	return findAffectedDeployments(ctx, orgID, customerOrgID, updateSecretValuePatchFunc(secretKey, newValue), nil)
}

func updateLicenseKeyPatchFunc(updatedLicenseKey types.LicenseKey) func([]types.LicenseKey) []types.LicenseKey {
	return func(licenseKeys []types.LicenseKey) []types.LicenseKey {
		patched := slices.Clone(licenseKeys)
		for i := range patched {
			if patched[i].ID == updatedLicenseKey.ID {
				patched[i] = updatedLicenseKey
				break
			}
		}
		return patched
	}
}

func findAffectedDeploymentsByLicenseKey(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID uuid.UUID,
	updatedLicenseKey types.LicenseKey,
) ([]api.AffectedDeployment, error) {
	return findAffectedDeployments(ctx, orgID, &customerOrgID, nil, updateLicenseKeyPatchFunc(updatedLicenseKey))
}

func findAffectedDeployments(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID *uuid.UUID,
	patchSecrets func([]types.SecretWithUpdatedBy) []types.SecretWithUpdatedBy,
	patchLicenseKeys func([]types.LicenseKey) []types.LicenseKey,
) ([]api.AffectedDeployment, error) {
	targets, err := db.GetDeploymentTargetsByScope(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	}
	secrets, err := db.GetSecretsByScope(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	} else if patchSecrets != nil {
		secrets = patchSecrets(secrets)
	}
	licenseKeys, err := db.GetLicenseKeysByScope(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	} else if patchLicenseKeys != nil {
		licenseKeys = patchLicenseKeys(licenseKeys)
	}

	affected := make([]api.AffectedDeployment, 0)
	for _, target := range targets {
		deployments, err := db.GetDeploymentsForDeploymentTarget(ctx, target.ID)
		if err != nil {
			return nil, err
		}

		for _, deployment := range deployments {
			if len(deployment.ValuesHash) != sha256.Size {
				continue
			}
			newHash, err := deploymentvalues.RenderAndHash(&deployment, secrets, licenseKeys)
			if err != nil {
				return nil, err
			}
			if !bytes.Equal(newHash[:], deployment.ValuesHash) {
				affected = append(affected, api.AffectedDeployment{
					DeploymentTargetID:   target.ID,
					DeploymentTargetName: target.Name,
					DeploymentID:         deployment.ID,
					ApplicationName:      deployment.Application.Name,
				})
			}
		}
	}

	return affected, nil
}

func deleteSecretPatchFunc(secretID uuid.UUID) func([]types.SecretWithUpdatedBy) []types.SecretWithUpdatedBy {
	return func(secrets []types.SecretWithUpdatedBy) []types.SecretWithUpdatedBy {
		return slices.DeleteFunc(slices.Clone(secrets), func(s types.SecretWithUpdatedBy) bool {
			return s.ID == secretID
		})
	}
}

func findDeploymentsReferencingSecret(
	ctx context.Context,
	orgID uuid.UUID,
	secretID uuid.UUID,
	customerOrgID *uuid.UUID,
) ([]api.AffectedDeployment, error) {
	return findReferencingDeployments(ctx, orgID, customerOrgID, deleteSecretPatchFunc(secretID), nil)
}

func deleteLicenseKeyPatchFunc(licenseKeyID uuid.UUID) func([]types.LicenseKey) []types.LicenseKey {
	return func(licenseKeys []types.LicenseKey) []types.LicenseKey {
		return slices.DeleteFunc(slices.Clone(licenseKeys), func(lk types.LicenseKey) bool {
			return lk.ID == licenseKeyID
		})
	}
}

func findDeploymentsReferencingLicenseKey(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID uuid.UUID,
	licenseKeyID uuid.UUID,
) ([]api.AffectedDeployment, error) {
	return findReferencingDeployments(ctx, orgID, &customerOrgID, nil, deleteLicenseKeyPatchFunc(licenseKeyID))
}

func findReferencingDeployments(
	ctx context.Context,
	orgID uuid.UUID,
	customerOrgID *uuid.UUID,
	patchSecrets func([]types.SecretWithUpdatedBy) []types.SecretWithUpdatedBy,
	patchLicenseKeys func([]types.LicenseKey) []types.LicenseKey,
) ([]api.AffectedDeployment, error) {
	targets, err := db.GetDeploymentTargetsByScope(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	}
	secrets, err := db.GetSecretsByScope(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	} else if patchSecrets != nil {
		secrets = patchSecrets(secrets)
	}
	licenseKeys, err := db.GetLicenseKeysByScope(ctx, orgID, customerOrgID)
	if err != nil {
		return nil, err
	} else if patchLicenseKeys != nil {
		licenseKeys = patchLicenseKeys(licenseKeys)
	}

	referencing := make([]api.AffectedDeployment, 0)
	for _, target := range targets {
		deployments, err := db.GetDeploymentsForDeploymentTarget(ctx, target.ID)
		if err != nil {
			return nil, err
		}

		for _, deployment := range deployments {
			_, err := deploymentvalues.RenderAndHash(&deployment, secrets, licenseKeys)
			if errors.Is(err, deploymentvalues.ErrInvalidTemplate) {
				referencing = append(referencing, api.AffectedDeployment{
					DeploymentTargetID:   target.ID,
					DeploymentTargetName: target.Name,
					DeploymentID:         deployment.ID,
					ApplicationName:      deployment.Application.Name,
				})
			} else if err != nil {
				return nil, err
			}
		}
	}

	return referencing, nil
}

func triggerAffectedDeployments(ctx context.Context, affected []api.AffectedDeployment) error {
	createdByUserID := auth.Authentication.Require(ctx).CurrentUserID()

	byTarget := make(map[uuid.UUID][]api.AffectedDeployment)
	for _, ad := range affected {
		byTarget[ad.DeploymentTargetID] = append(byTarget[ad.DeploymentTargetID], ad)
	}

	for targetID, affectedForTarget := range byTarget {
		target, err := db.GetDeploymentTarget(ctx, targetID, nil, nil)
		if err != nil {
			return err
		}
		secrets, err := db.GetSecretsForDeploymentTarget(ctx, target.DeploymentTarget)
		if err != nil {
			return err
		}
		licenseKeys, err := db.GetLicenseKeysForDeploymentTarget(ctx, target.DeploymentTarget)
		if err != nil {
			return err
		}

		for _, affectedDeployment := range affectedForTarget {
			index := slices.IndexFunc(target.Deployments, func(d types.DeploymentWithLatestRevision) bool {
				return d.ID == affectedDeployment.DeploymentID
			})
			if index < 0 {
				return fmt.Errorf("deployment %s not found for deployment target %s",
					affectedDeployment.DeploymentID, affectedDeployment.DeploymentTargetID)
			}
			request := deploymentRequestFromLatestRevision(target.Deployments[index])
			request.CreatedByUserAccountID = &createdByUserID
			if err := setDeploymentRequestValuesHash(&request, secrets, licenseKeys); err != nil {
				return err
			}
			if _, err := db.CreateDeploymentRevision(ctx, &request); err != nil {
				return err
			}
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
