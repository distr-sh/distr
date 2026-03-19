// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package registry implements a docker V2 registry and the OCI distribution specification.
//
// It is designed to be used anywhere a low dependency container registry is needed, with an
// initial focus on tests.
//
// Its goal is to be standards compliant and its strictness will increase over time.
//
// This is currently a low flightmiles system. It's likely quite safe to use in tests; If you're using it
// in production, please let us know how and send us CL's for integration tests.
package registry

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"slices"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/authn/authinfo"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/registry/audit"
	"github.com/distr-sh/distr/internal/registry/authz"
	"github.com/distr-sh/distr/internal/registry/blob"
	"github.com/distr-sh/distr/internal/registry/blob/s3"
	"github.com/distr-sh/distr/internal/registry/manifest"
	"github.com/distr-sh/distr/internal/registry/manifest/db"
	"github.com/getsentry/sentry-go"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type registry struct {
	log              *zap.SugaredLogger
	blobs            blobs
	manifests        manifests
	referrersEnabled bool
	warnings         map[float64]string
	middlewares      []func(http.Handler) http.Handler
}

// https://docs.docker.com/registry/spec/api/#api-version-check
// https://github.com/opencontainers/distribution-spec/blob/master/spec.md#api-version-check
func (r *registry) v2(resp http.ResponseWriter, req *http.Request) *regError {
	if r.warnings != nil {
		rnd := rand.Float64()
		for prob, msg := range r.warnings {
			if prob > rnd {
				resp.Header().Add("Warning", fmt.Sprintf(`299 - "%s"`, msg))
			}
		}
	}

	if isBlob(req) {
		return r.blobs.handle(resp, req)
	}
	if isManifest(req) {
		return r.manifests.handle(resp, req)
	}
	if isTags(req) {
		return r.manifests.handleTags(resp, req)
	}
	if isCatalog(req) {
		return r.manifests.handleCatalog(resp, req)
	}
	if r.referrersEnabled && isReferrers(req) {
		return r.manifests.handleReferrers(resp, req)
	}
	resp.Header().Set("Docker-Distribution-API-Version", "registry/2.0")
	if req.URL.Path != "/v2/" && req.URL.Path != "/v2" {
		return &regError{
			Status:  http.StatusNotFound,
			Code:    "METHOD_UNKNOWN",
			Message: "We don't understand your method + url",
		}
	}
	resp.WriteHeader(200)
	return nil
}

func (r *registry) root(resp http.ResponseWriter, req *http.Request) {
	if rerr := r.v2(resp, req); rerr != nil {
		r.log.Warnf("%s %s %d %s %s", req.Method, req.URL, rerr.Status, rerr.Code, rerr.Message)
		if rerr.Status == http.StatusInternalServerError && rerr.Error != nil {
			sentry.GetHubFromContext(req.Context()).CaptureException(rerr.Error)
		}
		_ = rerr.Write(resp)
		return
	}
	r.log.Infof("%s %s", req.Method, req.URL)
}

// New returns a handler which implements the docker registry protocol.
// It should be registered at the site root.
func New(opts ...Option) http.Handler {
	reg := &registry{
		manifests: manifests{},
	}
	for _, o := range opts {
		o(reg)
	}
	var h http.Handler = http.HandlerFunc(reg.root)
	slices.Reverse(reg.middlewares)
	for _, mw := range reg.middlewares {
		h = mw(h)
	}
	return h
}

func NewDefault(
	ctx context.Context,
	logger *zap.Logger,
	pool *pgxpool.Pool,
	mailer mail.Mailer,
	tracer trace.TracerProvider,
) (http.Handler, error) {
	blobHandler, err := s3.NewBlobHandler(ctx)
	if err != nil {
		return nil, err
	}

	reg := New(
		WithLogger(logger),
		WithBlobHandler(blobHandler),
		WithManifestHandler(db.NewManifestHandler()),
		WithAuthorizer(authz.NewAuthorizer()),
		WithAuditor(audit.NewAuditor()),
		WithMiddlewares(
			chimiddleware.Recoverer,
			chimiddleware.RequestID,
			chimiddleware.RealIP,
			middleware.OTEL(tracer),
			middleware.Sentry,
			middleware.LoggerCtxMiddleware(logger),
			middleware.LoggingMiddleware,
			middleware.ContextInjectorMiddleware(pool, mailer, nil, nil),
			auth.ArtifactsAuthentication.Middleware,
			auth.ArtifactsAuthentication.ValidatorMiddleware(func(value authinfo.AuthInfoWithOrganization) error {
				if value.CurrentOrg() == nil {
					return errors.New("org is required")
				}
				return nil
			}),
		),
	)

	return reg, nil
}

// Option describes the available options
// for creating the registry.
type Option func(r *registry)

// WithLogger overrides the logger used to record requests to the registry.
func WithLogger(l *zap.Logger) Option {
	return func(r *registry) {
		r.log = l.Sugar()
		r.manifests.log = l.Sugar()
		r.blobs.log = l.Sugar()
	}
}

// WithReferrersSupport enables the referrers API endpoint (OCI 1.1+)
func WithReferrersSupport(enabled bool) Option {
	return func(r *registry) {
		r.referrersEnabled = enabled
	}
}

func WithWarning(prob float64, msg string) Option {
	return func(r *registry) {
		if r.warnings == nil {
			r.warnings = map[float64]string{}
		}
		r.warnings[prob] = msg
	}
}

func WithBlobHandler(h blob.BlobHandler) Option {
	return func(r *registry) {
		r.blobs.blobHandler = h
		r.manifests.blobHandler = h
	}
}

func WithManifestHandler(h manifest.ManifestHandler) Option {
	return func(r *registry) {
		r.manifests.manifestHandler = h
	}
}

func WithMiddlewares(m ...func(http.Handler) http.Handler) Option {
	return func(r *registry) {
		r.middlewares = append(r.middlewares, m...)
	}
}

func WithAuthorizer(a authz.Authorizer) Option {
	return func(r *registry) {
		r.blobs.authz = a
		r.manifests.authz = a
	}
}

func WithAuditor(a audit.ArtifactAuditor) Option {
	return func(r *registry) {
		r.manifests.audit = a
	}
}
