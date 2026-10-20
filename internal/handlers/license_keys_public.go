package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/distr-sh/distr/internal/licensekey"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/lestrrat-go/jwx/v4/jwk"
	"github.com/lestrrat-go/jwx/v4/jwk/jwkbb"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

func PublicLicenseKeysRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Licensing"))

	r.Get("/public-key", respond.Handler(getLicenseKeyPublicKeyHandler)).With(
		option.Description("Get the X.509/PEM encoded public key for verifying a license key"),
		option.Response(http.StatusOK, nil, option.ContentType("text/plain")),
	)
}

func getLicenseKeyPublicKeyHandler(r *http.Request) (respond.Responder, error) {
	if pk, err := licensekey.PublicKey(); err != nil {
		if errors.Is(err, licensekey.ErrNoSigningKey) {
			return licenseKeyPublicKeyNotFound, nil
		} else {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}
	} else if rawKey, err := jwk.Export[any](pk); err != nil {
		return nil, fmt.Errorf("failed to export public key: %w", err)
	} else if encodedKey, err := jwkbb.EncodePEM(rawKey); err != nil {
		return nil, fmt.Errorf("failed to encode public key: %w", err)
	} else {
		return respond.Stream("text/plain; charset=utf-8", bytes.NewReader(encodedKey)), nil
	}
}
