package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/deploymentvalues"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/resources"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func TutorialsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupHidden(true))
	r.Use(middleware.RequireOrgAndRole, middleware.RequireVendor, middleware.RequireAdmin)
	r.Get("/", respond.Handler(getTutorialProgresses))
	r.Route("/{tutorial}", func(r chiopenapi.Router) {
		r.Get("/", respond.Handler(getTutorialProgress))
		r.With(middleware.BlockSuperAdmin).Put("/", respond.Handler(saveTutorialProgress))
	})
}

func getTutorialProgresses(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	if progresses, err := db.GetTutorialProgresses(ctx, auth.CurrentUserID(), *auth.CurrentOrgID()); err != nil {
		return nil, fmt.Errorf("could not get tutorial progresses: %w", err)
	} else {
		return respond.JSON(progresses), nil
	}
}

func getTutorialProgress(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	tutorial := r.PathValue("tutorial")
	if progress, err := db.GetTutorialProgress(ctx, auth.CurrentUserID(), *auth.CurrentOrgID(),
		types.Tutorial(tutorial)); errors.Is(err, distrerr.ErrNotFound) {
		return tutorialProgressNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not get tutorial progress: %w", err)
	} else {
		return respond.JSON(progress), nil
	}
}

func saveTutorialProgress(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	tutorial := types.Tutorial(r.PathValue("tutorial"))

	var progress any
	if req, resp := decodeJSON[api.TutorialProgressRequest](r); resp != nil {
		return resp, nil
	} else if err := db.RunTx(ctx, func(ctx context.Context) error {
		if tutorial == types.TutorialControllers && req.StepID == "welcome" && req.TaskID == "start" {
			if dt, err := createSampleAppAndDeployment(ctx); err != nil {
				return fmt.Errorf("could not create sample app and deployment: %w", err)
			} else if dt != nil {
				// TODO save additional data?
				req.Value = map[string]string{
					"deploymentTargetId": dt.ID.String(),
				}
			}
		}

		var err error
		if progress, err = db.SaveTutorialProgress(
			ctx, auth.CurrentUserID(), *auth.CurrentOrgID(), tutorial, &req,
		); err != nil {
			return fmt.Errorf("could not save tutorial progress: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	} else {
		return respond.JSON(progress), nil
	}
}

func createSampleAppAndDeployment(ctx context.Context) (*types.DeploymentTargetFull, error) {
	if app, err := createHelloDistrApp(ctx); err != nil {
		return nil, fmt.Errorf("failed to create hello-distr app: %w", err)
	} else if dt, err := createHelloDistrDeploymentTarget(ctx); err != nil {
		return nil, fmt.Errorf("failed to create hello-distr deployment target: %w", err)
	} else if err := createHelloDistrDeploymentAndRevision(ctx, app.Versions[0].ID, dt.ID); err != nil {
		return nil, fmt.Errorf("failed to deploy hello-distr: %w", err)
	} else {
		return dt, nil
	}
}

func createHelloDistrApp(ctx context.Context) (*types.Application, error) {
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	application := types.Application{
		Name:               "hello-distr",
		Type:               types.DeploymentTypeDocker,
		VersioningStrategy: types.VersioningStrategySemver,
	}

	var composeFileData []byte
	var templateFileData []byte
	if composeFile, err := resources.Get("apps/hello-distr/docker-compose.yaml"); err != nil {
		log.Warn("failed to read hello-distr compose file", zap.Error(err))
	} else {
		composeFileData = composeFile
	}
	if templateFile, err := resources.Get("apps/hello-distr/template.env"); err != nil {
		log.Warn("failed to read hello-distr template file", zap.Error(err))
	} else {
		templateFileData = templateFile
	}

	version := types.ApplicationVersion{
		Name:             "0.4.13", // renovate: datasource=github-releases depName=distr-sh/hello-distr
		LinkTemplate:     "{{ .Env.HELLO_DISTR_PROTOCOL }}://{{ .Env.HELLO_DISTR_HOST }}",
		ComposeFileData:  composeFileData,
		TemplateFileData: templateFileData,
	}

	if err := db.CreateApplication(ctx, &application, *auth.CurrentOrgID()); err != nil {
		return nil, err
	}
	version.ApplicationID = application.ID
	if err := db.CreateApplicationVersion(ctx, &version); err != nil {
		return nil, err
	}

	application.Versions = append(application.Versions, version)
	return &application, nil
}

func createHelloDistrDeploymentTarget(ctx context.Context) (*types.DeploymentTargetFull, error) {
	auth := auth.Authentication.Require(ctx)
	dt := types.DeploymentTargetFull{
		DeploymentTarget: types.DeploymentTarget{
			Name:                    "hello-distr-tutorial",
			Type:                    types.DeploymentTypeDocker,
			AutohealEnabled:         true,
			MetricsEnabled:          true,
			DeploymentLogsEnabled:   true,
			AutomaticUpdatesEnabled: true,
		},
	}
	if controllerVersion, err := db.GetCurrentControllerVersion(ctx); err != nil {
		return nil, err
	} else {
		dt.ControllerVersionID = &controllerVersion.ID
		if err := db.CreateDeploymentTarget(
			ctx,
			&dt,
			*auth.CurrentOrgID(),
			auth.CurrentUserID(),
			auth.CurrentCustomerOrgID(),
		); err != nil {
			return nil, err
		} else {
			return &dt, nil
		}
	}
}

const helloDistrEnvironment = `
# mandatory values:
HELLO_DISTR_HOST=localhost
HELLO_DISTR_PROTOCOL=http
HELLO_DISTR_DB_NAME=hello-distr
HELLO_DISTR_DB_USER=distr
HELLO_DISTR_DB_PASSWORD=distr123
`

func createHelloDistrDeploymentAndRevision(ctx context.Context, appVersionID uuid.UUID, dtID uuid.UUID) error {
	deploymentRequest := &api.DeploymentRequest{
		ApplicationVersionID: appVersionID,
		DeploymentTargetID:   dtID,
		EnvFileData:          []byte(helloDistrEnvironment),
		DockerType:           util.PtrTo(types.DockerTypeCompose),
		Trigger:              types.DeploymentRevisionTriggerUser,
	}
	if err := db.CreateDeployment(ctx, deploymentRequest); err != nil {
		return err
	} else if hash, err := deploymentvalues.RenderAndHash(deploymentRequest, nil, nil); err != nil {
		return err
	} else {
		deploymentRequest.ValuesHash = hash[:]
		if _, err := db.CreateDeploymentRevision(ctx, deploymentRequest); err != nil {
			return err
		}
		return nil
	}
}
