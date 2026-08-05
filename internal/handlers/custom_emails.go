package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/custommail"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/mailtemplates"
	"github.com/distr-sh/distr/internal/mapping"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/go-mailx/mailx"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func CustomEmailsRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Custom Email"))
	r.Use(middleware.RequireVendor, middleware.RequireOrgAndRole, middleware.RequireAdmin)
	r.Get("/", getCustomEmailConfigurationHandler).
		With(option.Description("Get the email configuration of the current organization")).
		With(option.Response(http.StatusOK, api.CustomEmailConfiguration{}))
	r.With(middleware.BlockSuperAdmin).Group(func(r chiopenapi.Router) {
		r.Put("/", updateCustomEmailConfigurationHandler).
			With(option.Description("Create or update the email configuration of the current organization")).
			With(option.Request(api.UpdateCustomEmailConfigurationRequest{})).
			With(option.Response(http.StatusOK, api.CustomEmailConfiguration{}))
		r.Delete("/", deleteCustomEmailConfigurationHandler).
			With(option.Description("Delete the email configuration of the current organization"))
		r.Post("/test", testCustomEmailConfigurationHandler).
			With(option.Description("Send a test email to the current user with the given email configuration")).
			With(option.Request(api.TestCustomEmailConfigurationRequest{}))
	})
}

func getCustomEmailConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)
	config, err := db.GetCustomEmailConfiguration(ctx, *auth.CurrentOrgID())
	if errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		log.Error("failed to get custom email configuration", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.CustomEmailConfigurationToResponse(*config))
	}
}

func updateCustomEmailConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	request, err := JsonBody[api.UpdateCustomEmailConfigurationRequest](w, r)
	if err != nil {
		return
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, err := customEmailConfigurationFromRequest(ctx, request.CustomEmailSettings, *auth.CurrentOrgID())
	if err != nil {
		log.Error("failed to get stored custom email configuration", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	config.Enabled = request.Enabled
	config.UpdatedByUserAccountID = &auth.CurrentUser().ID

	if err := db.UpsertCustomEmailConfiguration(ctx, &config); err != nil {
		log.Error("failed to update custom email configuration", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		RespondJSON(w, mapping.CustomEmailConfigurationToResponse(config))
	}
}

func deleteCustomEmailConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	if err := db.DeleteCustomEmailConfiguration(ctx, *auth.CurrentOrgID()); errors.Is(err, apierrors.ErrNotFound) {
		http.NotFound(w, r)
	} else if err != nil {
		log.Error("failed to delete custom email configuration", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// Sending is the only real validation of a configuration, so the provider error is reported
// verbatim to tell the admin what to fix.
func testCustomEmailConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	auth := auth.Authentication.Require(ctx)

	request, err := JsonBody[api.TestCustomEmailConfigurationRequest](w, r)
	if err != nil {
		return
	}
	request.Normalize()
	if err := request.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, err := customEmailConfigurationFromRequest(ctx, request.CustomEmailSettings, *auth.CurrentOrgID())
	if err != nil {
		log.Error("failed to get stored custom email configuration", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mailer, err := custommail.MailerForConfiguration(config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := mailer.Send(ctx,
		mailx.To(auth.CurrentUser().Email),
		mailx.Subject("Distr email configuration test"),
		mailx.HtmlBodyTemplate(
			mailtemplates.CustomEmailTest(ctx, *auth.CurrentUser(), *auth.CurrentOrgWithBranding(), config),
		),
	); err != nil {
		log.Info("custom email configuration test failed", zap.Error(err))
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		w.WriteHeader(http.StatusNoContent)
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
		SMTPUsername:    settings.SMTPUsername,
		SMTPImplicitTLS: settings.SMTPImplicitTLS,
	}
	if settings.SMTPPassword != nil {
		config.SMTPPassword = *settings.SMTPPassword
		return config, nil
	}
	stored, err := db.GetCustomEmailConfiguration(ctx, orgID)
	if errors.Is(err, apierrors.ErrNotFound) {
		return config, nil
	} else if err != nil {
		return config, err
	}
	config.SMTPPassword = stored.SMTPPassword
	return config, nil
}
