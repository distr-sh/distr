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
	secretKeyPattern  = regexp.MustCompile(`^[a-zA-Z][\w_]*$`)
	errSecretNotFound = apierrors.NewNotFound("secret not found")
	errInvalidSecret  = apierrors.NewBadRequest("invalid secret ID")
)

func getSecrets(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	secrets, err := db.GetSecrets(ctx, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if err != nil {
		return nil, fmt.Errorf("failed to get secrets: %w", err)
	}
	return respond.JSON(mapping.List(secrets, mapping.SecretToAPI)), nil
}

func createSecret(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	body, err := decodeJSON[api.CreateSecretRequest](r)
	if err != nil {
		return nil, err
	}

	if body.Key == "" {
		return nil, apierrors.NewBadRequest("key is required")
	}
	if !secretKeyPattern.MatchString(body.Key) {
		return nil, apierrors.NewBadRequest("invalid key format")
	}
	if body.Value == "" {
		return nil, apierrors.NewBadRequest("value is required")
	}

	customerOrganizationID, err := customerScope(ctx, body.CustomerOrganizationID)
	if err != nil {
		return nil, err
	}

	secret, err := db.CreateSecret(
		ctx,
		*auth.CurrentOrgID(),
		customerOrganizationID,
		auth.CurrentUserID(),
		body.Key,
		dbcrypto.String(body.Value),
	)
	if errors.Is(err, apierrors.ErrConflict) {
		return nil, apierrors.NewConflict("secret with this key already exists")
	} else if err != nil {
		return nil, fmt.Errorf("failed to create secret: %w", err)
	}
	return respond.JSON(mapping.SecretToAPI(*secret)).WithStatus(http.StatusCreated), nil
}

func updateSecret(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	id, err := uuid.Parse(r.PathValue("secretId"))
	if err != nil {
		return nil, errInvalidSecret
	}

	body, err := decodeJSON[api.UpdateSecretRequest](r)
	if err != nil {
		return nil, err
	}

	existing, err := db.GetSecretByID(
		ctx,
		id,
		*auth.CurrentOrgID(),
		auth.CurrentCustomerOrgID(),
		auth.CurrentPartnerOrgID(),
	)
	if errors.Is(err, apierrors.ErrNotFound) {
		return nil, errSecretNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	affected, err := findAffectedDeploymentsBySecret(
		ctx,
		*auth.CurrentOrgID(),
		existing.Key,
		dbcrypto.String(body.Value),
		existing.CustomerOrganizationID,
	)
	if err != nil {
		return nil, wrapDeploymentValuesError(err, "invalid deployment values")
	}

	confirm := r.URL.Query().Get("confirm") == "true"
	if !confirm && len(affected) > 0 {
		return respond.JSON(api.AffectedDeploymentsConflictResponse{AffectedDeployments: affected}).
			WithStatus(http.StatusConflict), nil
	}

	var secret *types.SecretWithUpdatedBy
	err = db.RunTx(ctx, func(ctx context.Context) error {
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
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update secret: %w", err)
	}

	return respond.JSON(api.UpdateSecretResponse{
		SecretWithoutValue:  *mapping.SecretToAPI(*secret),
		AffectedDeployments: affected,
	}), nil
}

func deleteSecret(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	id, err := uuid.Parse(r.PathValue("secretId"))
	if err != nil {
		return nil, errInvalidSecret
	}

	existing, err := db.GetSecretByID(
		ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		return nil, errSecretNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	referencing, err := findDeploymentsReferencingSecret(ctx, *auth.CurrentOrgID(), id, existing.CustomerOrganizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to check deployment references for secret: %w", err)
	}
	if len(referencing) > 0 {
		return respond.JSON(api.AffectedDeploymentsConflictResponse{AffectedDeployments: referencing}).
			WithStatus(http.StatusConflict), nil
	}

	err = db.DeleteSecret(ctx, id, *auth.CurrentOrgID(), auth.CurrentCustomerOrgID(), auth.CurrentPartnerOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		return nil, errSecretNotFound
	} else if err != nil {
		return nil, fmt.Errorf("failed to delete secret: %w", err)
	}
	return respond.NoContent(), nil
}
