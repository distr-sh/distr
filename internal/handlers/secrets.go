package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func SecretsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Secrets"))

	r.Use(middleware.RequireOrgAndRole)

	r.Get("/", respond.Handler(getSecrets)).
		With(option.Description("List all secrets")).
		With(option.Response(http.StatusOK, []api.SecretWithoutValue{}))

	r.Group(func(r chiopenapi.Router) {
		r.Use(middleware.RequireReadWriteOrAdmin, middleware.BlockSuperAdmin)

		r.Post("/", respond.Handler(createSecret)).
			With(option.Description("Create a secret")).
			With(option.Request(api.CreateSecretRequest{})).
			With(option.Response(http.StatusCreated, api.SecretWithoutValue{}))

		r.Route("/{secretId}", func(r chiopenapi.Router) {
			r.Put("/", respond.Handler(updateSecret)).
				With(option.Description("Update a secret")).
				With(option.Request(api.UpdateSecretRequest{})).
				With(option.Response(http.StatusOK, api.UpdateSecretResponse{})).
				With(option.Response(http.StatusConflict, api.AffectedDeploymentsConflictResponse{}))

			r.Delete("/", respond.Handler(deleteSecret)).
				With(option.Description("Delete a secret")).
				With(option.Request(api.DeleteSecretRequest{})).
				With(option.Response(http.StatusConflict, api.AffectedDeploymentsConflictResponse{}))
		})
	})
}

var (
	secretKeyPattern = regexp.MustCompile(`^[a-zA-Z][\w_]*$`)
	secretNotFound   = respond.NotFound("secret not found")
	invalidSecretID  = respond.BadRequest("invalid secret ID")
)

func getSecrets(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if secrets, err := db.GetSecrets(
		ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	} else {
		return respond.JSON(mapping.List(secrets, mapping.SecretToAPI)), nil
	}
}

func createSecret(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if body, resp := decodeJSON[api.CreateSecretRequest](r); resp != nil {
		return resp, nil
	} else if body.Key == "" {
		return respond.BadRequest("key is required"), nil
	} else if !secretKeyPattern.MatchString(body.Key) {
		return respond.BadRequest("invalid key format"), nil
	} else if body.Value == "" {
		return respond.BadRequest("value is required"), nil
	} else if customerOrganizationID, resp, err := customerScope(
		ctx, body.CustomerOrganizationID,
	); resp != nil || err != nil {
		return resp, err
	} else if secret, err := db.CreateSecret(
		ctx,
		*auth.CurrentOrgID(),
		customerOrganizationID,
		auth.CurrentUserID(),
		body.Key,
		dbcrypto.String(body.Value),
	); errors.Is(err, apierrors.ErrConflict) {
		return respond.Conflict("secret with this key already exists"), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	} else {
		return respond.JSON(mapping.SecretToAPI(*secret)).WithStatus(http.StatusCreated), nil
	}
}

func updateSecret(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if id, err := uuid.Parse(r.PathValue("secretId")); err != nil {
		return invalidSecretID, nil
	} else if body, resp := decodeJSON[api.UpdateSecretRequest](r); resp != nil {
		return resp, nil
	} else if existing, err := db.GetSecretByID(
		ctx,
		id,
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
		auth.CurrentPartnerOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return secretNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	} else if affected, err := findAffectedDeploymentsBySecret(
		ctx,
		*auth.CurrentOrgID(),
		existing.Key,
		dbcrypto.String(body.Value),
		existing.CustomerOrganizationID,
	); errors.Is(err, deploymentvalues.ErrInvalidTemplate) {
		return respond.BadRequest(fmt.Sprintf("invalid deployment values: %v", err)), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to find deployments affected by secret: %w", err)
	} else if confirm := r.URL.Query().Get("confirm") == "true"; !confirm && len(affected) > 0 {
		return respond.JSON(api.AffectedDeploymentsConflictResponse{AffectedDeployments: affected}).
			WithStatus(http.StatusConflict), nil
	} else {
		var secret *types.SecretWithUpdatedBy
		if err := db.RunTx(ctx, func(ctx context.Context) error {
			var err error
			secret, err = db.UpdateSecret(
				ctx,
				id,
				existing.OrganizationID,
				existing.CustomerOrganizationID,
				auth.CurrentUserID(),
				dbcrypto.String(body.Value),
			)
			if err != nil {
				return err
			}
			return triggerAffectedDeployments(
				ctx, affected, new(auth.CurrentUserID()), types.DeploymentRevisionTriggerSecretChange,
			)
		}); err != nil {
			return nil, fmt.Errorf("failed to update secret: %w", err)
		} else {
			return respond.JSON(api.UpdateSecretResponse{
				SecretWithoutValue:  *mapping.SecretToAPI(*secret),
				AffectedDeployments: affected,
			}), nil
		}
	}
}

func deleteSecret(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if id, err := uuid.Parse(r.PathValue("secretId")); err != nil {
		return invalidSecretID, nil
	} else if existing, err := db.GetSecretByID(
		ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return secretNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	} else if referencing, err := findDeploymentsReferencingSecret(
		ctx, *auth.CurrentOrgID(), id, existing.CustomerOrganizationID,
	); err != nil {
		return nil, fmt.Errorf("failed to check deployment references for secret: %w", err)
	} else if len(referencing) > 0 {
		return respond.JSON(api.AffectedDeploymentsConflictResponse{AffectedDeployments: referencing}).
			WithStatus(http.StatusConflict), nil
	} else if err := db.DeleteSecret(
		ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID(),
	); errors.Is(err, apierrors.ErrNotFound) {
		return secretNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to delete secret: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}
