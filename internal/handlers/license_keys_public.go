package handlers

import (
	"errors"
	"net/http"

	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/getsentry/sentry-go"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func PublicLicenseKeysRouter(r chiopenapi.Router) {
	r.Get("/public-key", getLicenseKeyPublicKeyHandler()).With(
		option.Description("Get the X.509/PEM encoded public key for verifying a license key"),
		option.Tags("Licensing"),
		option.Response(http.StatusOK, nil, option.ContentType("text/plain")),
	)
}

func getLicenseKeyPublicKeyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		if pk, err := licensekey.PublicKey(); err != nil {
			if errors.Is(err, licensekey.ErrNoSigningKey) {
				http.NotFound(w, r)
			} else {
				log.Warn("failed to get public key", zap.Error(err))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		} else if encodedKey, err := jwk.EncodePEM(pk); err != nil {
			log.Warn("failed to encode public key", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			sentry.GetHubFromContext(ctx).CaptureException(err)
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write(encodedKey)
		}
	}
}
