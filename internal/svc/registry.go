package svc

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/distr-sh/distr/internal/buildconfig"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/jobs"
	"github.com/distr-sh/distr/internal/migrations"
	"github.com/distr-sh/distr/internal/oidc"
	distrprometheus "github.com/distr-sh/distr/internal/prometheus"
	"github.com/distr-sh/distr/internal/registry"
	"github.com/distr-sh/distr/internal/routing"
	"github.com/distr-sh/distr/internal/server"
	"github.com/distr-sh/distr/internal/tracers"
	"github.com/go-chi/chi/v5"
	"github.com/go-mailx/mailx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type Registry struct {
	dbPool            *pgxpool.Pool
	logger            *zap.Logger
	mailer            *mailx.Mailer
	execDbMigrations  bool
	artifactsRegistry http.Handler
	tracers           *tracers.Tracers
	jobsScheduler     *jobs.Scheduler
	oidcer            *oidc.OIDCer
	promRegistry      *prometheus.Registry
	promCollector     *distrprometheus.DistrCollector
	s3Client          *s3.Client
}

func New(ctx context.Context, options ...RegistryOption) (*Registry, error) {
	var reg Registry
	for _, opt := range options {
		opt(&reg)
	}
	return newRegistry(ctx, &reg)
}

func NewDefault(ctx context.Context) (*Registry, error) {
	var reg Registry
	return newRegistry(ctx, &reg)
}

func newRegistry(ctx context.Context, reg *Registry) (*Registry, error) {
	reg.logger = createLogger()

	ctx = internalctx.WithLogger(ctx, reg.logger)

	reg.logger.Info("initializing service registry",
		zap.String("version", buildconfig.Version()),
		zap.String("commit", buildconfig.Commit()),
		zap.String("edition", buildconfig.Edition()),
		zap.Bool("release", buildconfig.IsRelease()))

	reg.promCollector = distrprometheus.NewDistrCollector()
	reg.promRegistry = createPrometheusRegistry(reg.promCollector)

	if tracers, err := reg.createTracer(ctx); err != nil {
		return nil, err
	} else {
		reg.tracers = tracers
	}

	if mailer, err := createMailer(ctx); err != nil {
		return nil, err
	} else {
		reg.mailer = mailer
	}

	if reg.execDbMigrations {
		if err := migrations.Up(reg.logger); err != nil {
			return nil, err
		}
	}

	if db, err := reg.createDBPool(ctx); err != nil {
		return nil, err
	} else {
		reg.dbPool = db
	}

	if env.RegistryEnabled() {
		reg.s3Client = newS3Client(ctx)
	}

	if scheduler, err := reg.createJobsScheduler(); err != nil {
		return nil, err
	} else {
		reg.jobsScheduler = scheduler
	}

	if artifactRegistry, err := reg.createArtifactsRegistry(ctx); err != nil {
		return nil, err
	} else {
		reg.artifactsRegistry = artifactRegistry
	}

	if oidcer, err := reg.createOIDCer(ctx, reg.logger); err != nil {
		return nil, err
	} else {
		reg.oidcer = oidcer
	}

	return reg, nil
}

func (r *Registry) Shutdown(ctx context.Context) error {
	if err := r.jobsScheduler.Shutdown(); err != nil {
		r.logger.Warn("job scheduler shutdown failed", zap.Error(err))
	}

	r.logger.Warn("shutting down database connections")
	r.dbPool.Close()

	if err := r.tracers.Shutdown(ctx); err != nil {
		r.logger.Warn("tracer shutdown failed", zap.Error(err))
	}

	// some devices like stdout and stderr can not be synced by the OS
	if err := r.logger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
		return fmt.Errorf("logger sync failed: %w", err)
	}

	return nil
}

func (reg *Registry) createArtifactsRegistry(ctx context.Context) (http.Handler, error) {
	syncer := NewUpstreamSyncer()
	return registry.NewDefault(
		ctx,
		reg.GetLogger().With(zap.String("component", "registry")),
		reg.GetDbPool(),
		reg.GetMailer(),
		reg.GetTracers().Registry(),
		reg.s3Client,
		syncer,
		syncer,
	)
}

func (r *Registry) GetRouter() http.Handler {
	return routing.NewRouter(
		r.GetLogger(),
		r.GetDbPool(),
		r.GetMailer(),
		r.GetTracers(),
		r.GetOIDCer(),
		r.GetPrometheusCollector(),
		NewUpstreamSyncer(),
	)
}

func (r *Registry) GetArtifactsRouter() http.Handler {
	return r.artifactsRegistry
}

func (r *Registry) GetMetricsRouter() http.Handler {
	m := chi.NewMux()

	if metricsToken := env.MetricsBearerToken(); metricsToken != nil {
		expectedToken := []byte(*metricsToken)
		m.Use(func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authorization := r.Header.Get("Authorization")
				if strings.HasPrefix(authorization, "Bearer ") {
					providedToken := []byte(authorization[len("Bearer "):])
					if subtle.ConstantTimeCompare(expectedToken, providedToken) == 1 {
						h.ServeHTTP(w, r)
						return
					}
				}

				w.Header().Set("WWW-Authenticate", "Bearer")
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			})
		})
	}

	h := promhttp.HandlerFor(r.promRegistry, promhttp.HandlerOpts{})
	m.Get("/metrics", h.ServeHTTP)
	return m
}

func (r *Registry) GetServer() server.Server {
	return server.NewServer(r.GetRouter(), r.logger.With(zap.String("server", "main")))
}

func (r *Registry) GetArtifactsServer() server.Server {
	if env.RegistryEnabled() {
		return server.NewServer(r.GetArtifactsRouter(), r.logger.With(zap.String("server", "registry")))
	} else {
		return server.NewNoop()
	}
}

func (r *Registry) GetMetricsServer() server.Server {
	if env.MetricsEnabled() {
		return server.NewServer(r.GetMetricsRouter(), r.logger.With(zap.String("server", "metrics")))
	} else {
		return server.NewNoop()
	}
}

func (r *Registry) GetPrometheusCollector() *distrprometheus.DistrCollector {
	return r.promCollector
}

func (r *Registry) GetS3Client() *s3.Client {
	return r.s3Client
}
