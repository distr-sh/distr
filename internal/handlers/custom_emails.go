package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/custommail"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/dbcrypto"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func CustomEmailsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Custom Email"))
	r.Use(middleware.RequireVendor, middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", respond.Handler(getCustomEmailConfigurationHandler)).
		With(option.Description("Get the email configuration of the current organization")).
		With(option.Response(http.StatusOK, api.CustomEmailConfiguration{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Put("/", respond.Handler(updateCustomEmailConfigurationHandler)).
			With(option.Description("Create or update the email configuration of the current organization")).
			With(option.Request(api.UpdateCustomEmailConfigurationRequest{})).
			With(option.Response(http.StatusOK, api.CustomEmailConfiguration{}))
		r.Delete("/", respond.Handler(deleteCustomEmailConfigurationHandler)).
			With(option.Description("Delete the email configuration of the current organization"))
		r.Post("/test", respond.Handler(testCustomEmailConfigurationHandler)).
			With(option.Description("Send a test email to the current user with the given email configuration")).
			With(option.Request(api.TestCustomEmailConfigurationRequest{}))
	})
}

var customEmailConfigurationNotFound = respond.NotFound("email configuration not found")

func getCustomEmailConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)
	config, err := db.GetCustomEmailConfiguration(ctx, *auth.CurrentOrgID())
	if errors.Is(err, distrerr.ErrNotFound) {
		return customEmailConfigurationNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get custom email configuration: %w", err)
	} else {
		return respond.JSON(mapping.CustomEmailConfigurationToAPI(*config)), nil
	}
}

func updateCustomEmailConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	request, resp := decodeJSON[api.UpdateCustomEmailConfigurationRequest](r)
	if resp != nil {
		return resp, nil
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if config, err := customEmailConfigurationFromRequest(
		ctx, request.CustomEmailSettings, *auth.CurrentOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get stored custom email configuration: %w", err)
	} else {
		config.Enabled = request.Enabled
		config.UpdatedByUserAccountID = &auth.CurrentUser().ID

		if err := db.UpsertCustomEmailConfiguration(ctx, &config); err != nil {
			return nil, fmt.Errorf("failed to update custom email configuration: %w", err)
		} else {
			return respond.JSON(mapping.CustomEmailConfigurationToAPI(config)), nil
		}
	}
}

func deleteCustomEmailConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if err := db.DeleteCustomEmailConfiguration(ctx, *auth.CurrentOrgID()); errors.Is(err, distrerr.ErrNotFound) {
		return customEmailConfigurationNotFound, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to delete custom email configuration: %w", err)
	} else {
		return respond.NoContent(), nil
	}
}

// Sending is the only real validation of a configuration, so the provider error is reported
// verbatim to tell the admin what to fix.
func testCustomEmailConfigurationHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	request, resp := decodeJSON[api.TestCustomEmailConfigurationRequest](r)
	if resp != nil {
		return resp, nil
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if config, err := customEmailConfigurationFromRequest(
		ctx, request.CustomEmailSettings, *auth.CurrentOrgID(),
	); err != nil {
		return nil, fmt.Errorf("failed to get stored custom email configuration: %w", err)
	} else if mailer, err := custommail.MailerForConfiguration(config); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else if err := mailer.Send(ctx,
		mailx.To(auth.CurrentUser().Email),
		mailx.Subject("Distr email configuration test"),
		mailx.HtmlBodyTemplate(
			mailtemplates.CustomEmailTest(ctx, *auth.CurrentUser(), *auth.CurrentOrgWithBranding(), config),
		),
	); err != nil {
		log.Info("custom email configuration test failed", zap.Error(err))
		return respond.BadRequest(err.Error()), nil
	} else {
		return respond.NoContent(), nil
	}
}

// An omitted password keeps the stored one, so that the password never has to be sent to the
// client and re-entered to change another setting.
func customEmailConfigurationFromRequest(
	ctx context.Context,
	settings api.CustomEmailSettings,
	orgID uuid.UUID,
) (types.CustomEmailConfiguration, error) {
	config := types.CustomEmailConfiguration{
		OrganizationID:  orgID,
		FromAddress:     settings.FromAddress,
		SMTPHost:        settings.SMTPHost,
		SMTPPort:        settings.SMTPPort,
		SMTPUsername:    dbcrypto.String(settings.SMTPUsername),
		SMTPImplicitTLS: settings.SMTPImplicitTLS,
	}
	if settings.SMTPPassword != nil {
		config.SMTPPassword = dbcrypto.String(*settings.SMTPPassword)
		return config, nil
	}
	stored, err := db.GetCustomEmailConfiguration(ctx, orgID)
	if errors.Is(err, distrerr.ErrNotFound) {
		return config, nil
	} else if err != nil {
		return config, err
	}
	config.SMTPPassword = stored.SMTPPassword
	return config, nil
}
