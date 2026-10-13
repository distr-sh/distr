package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/respond"
)

func getDeploymentTargetNotesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)

	if notes, err := db.GetDeploymentTargetNotes(ctx, deploymentTarget.ID, auth.CurrentCustomerOrgID()); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			// empty notes response
			return respond.JSON(api.DeploymentTargetNotes{}), nil
		} else {
			return nil, fmt.Errorf("failed to get deployment target notes: %w", err)
		}
	} else {
		return respond.JSON(mapping.DeploymentTargetNotesToAPI(*notes)), nil
	}
}

func putDeploymentTargetNotesHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	deploymentTarget := internalctx.GetDeploymentTarget(ctx)

	if request, resp := decodeJSON[api.DeploymentTargetNotesRequest](r); resp != nil {
		return resp, nil
	} else if notes, err := db.CreateOrUpdateDeploymentTargetNotes(
		ctx,
		deploymentTarget.ID,
		auth.CurrentCustomerOrgID(),
		auth.CurrentUserID(),
		request.Notes,
	); err != nil {
		return nil, fmt.Errorf("failed to create or update deployment target notes: %w", err)
	} else {
		return respond.JSON(mapping.DeploymentTargetNotesToAPI(*notes)), nil
	}
}
