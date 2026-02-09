package routing

import (
	"net/http"
	"time"

	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/frontend"
	"github.com/distr-sh/distr/internal/handlers"
	"github.com/distr-sh/distr/internal/mail"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/oidc"
	"github.com/distr-sh/distr/internal/tracers"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oaswrap/spec-ui/config"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/openapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

const apiDescription = `
Distr enables software and AI companies to distribute applications to self-managed customers with minimal setup.

## Main features

- **Centralized Management:** View & manage all deployments, artifacts, connected agents, self-managed &
  BYOC customers via the intuitive web UI
- **Deployment Automation:** Optional prebuilt Helm and Docker agents manage deployments, collect logs and metrics,
  and allow remote troubleshooting.
- **White-label customer portal:** Let your customers control their deployments or download your artifacts
- **License Management:** Distribute specific versions of your application to specific customers
- **Container registry:** Distribute OCI-compatible artifacts (Docker images, Helm charts, Terraform modules)
  with built-in granular access control and analytics.
- Access the API using our [**rich SDK**](https://distr.sh/docs/integrations/sdk/)
- Fully Open Source and [self-hostable](https://distr.sh/docs/self-hosting/getting-started/)

Check out the hosted version at https://distr.sh/get-started/.

## About

Distr is an Open Source software distribution platform that provides a ready-to-use setup with prebuilt components to
help software and AI companies distribute applications to customers in complex, self-managed environments.

**Use cases include:**

- On-premises, VPC and self-managed software deployments
- Bring Your Own Cloud (BYOC) automation
- Edge & Fleet management

Read more about Distr and our use cases at https://distr.sh/docs/
`

func NewRouter(
	logger *zap.Logger, db *pgxpool.Pool, mailer mail.Mailer, tracers *tracers.Tracers, oidcer *oidc.OIDCer,
) http.Handler {
	baseRouter := chi.NewRouter()
	baseRouter.Use(
		// Handles panics
		chimiddleware.Recoverer,
		// Reject bodies larger than 1MiB
		chimiddleware.RequestSize(1048576),
	)

	openapiRouter := chiopenapi.NewRouter(
		baseRouter,
		option.WithTitle("Distr API Reference"),
		option.WithDescription(apiDescription),
		option.WithVersion(buildconfig.Version()),
		option.WithSecurity("bearer", option.SecurityHTTPBearer("Bearer")),
		option.WithSecurity(
			"accessToken",
			option.SecurityAPIKey("Authorization", openapi.SecuritySchemeAPIKeyInHeader),
			option.SecurityDescription(
				"Provide a PAT using the Authorization header and adding the AccessToken prefix.\n\n"+
					"Example: `Authorization: AccessToken distr-xxxxxx`",
			),
		),
		option.WithStoplightElements(config.StoplightElements{
			HideSchemas: true,
			Logo:        "/distr-logo.svg",
			Layout:      "responsive",
		}),
	)
	openapiRouter.Route("/api", ApiRouter(logger, db, mailer, tracers, oidcer))

	baseRouter.Mount("/internal", InternalRouter())
	baseRouter.Mount("/status", StatusRouter())
	baseRouter.Mount("/ready", ReadyRouter(db))
	baseRouter.Mount("/.well-known", WellKnownRouter())
	baseRouter.Mount("/", FrontendRouter())

	return baseRouter
}

func ApiRouter(
	logger *zap.Logger,
	db *pgxpool.Pool,
	mailer mail.Mailer,
	tracers *tracers.Tracers,
	oidcer *oidc.OIDCer,
) func(r chiopenapi.Router) {
	return func(r chiopenapi.Router) {
		r.Use(
			chimiddleware.RequestID,
			chimiddleware.RealIP,
			middleware.Sentry,
			middleware.LoggerCtxMiddleware(logger),
			middleware.LoggingMiddleware,
			middleware.ContextInjectorMiddleware(db, mailer, oidcer),
		)

		r.Route("/v1", func(r chiopenapi.Router) {
			r.Group(func(r chiopenapi.Router) {
				r.Use(
					middleware.OTEL(tracers.Default()),
				)

				// public routes go here
				r.Group(func(r chiopenapi.Router) {
					r.Route("/auth", handlers.AuthRouter)
					r.Route("/webhook", handlers.WebhookRouter)
				})

				// authenticated routes go here
				r.Group(func(r chiopenapi.Router) {
					r.WithOptions(
						option.GroupSecurity("accessToken"),
						option.GroupSecurity("bearer"),
					)

					r.Use(
						middleware.SentryUser,
						auth.Authentication.Middleware,
						httprate.Limit(30, 1*time.Second, httprate.WithKeyFuncs(middleware.RateLimitUserIDKey)),
						httprate.Limit(60, 1*time.Minute, httprate.WithKeyFuncs(middleware.RateLimitUserIDKey)),
						httprate.Limit(2000, 1*time.Hour, httprate.WithKeyFuncs(middleware.RateLimitUserIDKey)),

						// TODO (low-prio) in the future, additionally check token audience and require it to be "api"/"user",
						// such that agents cant access anything here (they also can't now, because their tokens will not
						// pass the Authentication chain (DbAuthenticator can't find the user -> 401)
					)
					r.Route("/agent-versions", handlers.AgentVersionsRouter)
					r.Route("/application-licenses", handlers.ApplicationLicensesRouter)
					r.Route("/applications", handlers.ApplicationsRouter)
					r.Route("/artifact-licenses", handlers.ArtifactLicensesRouter)
					r.Route("/artifact-pulls", handlers.ArtifactPullsRouter)
					r.Route("/artifacts", handlers.ArtifactsRouter)
					r.Route("/billing", handlers.BillingRouter)
					r.Route("/context", handlers.ContextRouter)
					r.Route("/customer-organizations", handlers.CustomerOrganizationsRouter)
					r.Route("/dashboard", handlers.DashboardRouter)
					r.Route("/alert-configurations", handlers.AlertConfigurationsRouter)
					r.Route("/deployment-target-metrics", handlers.DeploymentTargetMetricsRouter)
					r.Route("/deployment-targets", handlers.DeploymentTargetsRouter)
					r.Route("/deployments", handlers.DeploymentsRouter)
					r.Route("/files", handlers.FileRouter)
					r.Route("/notification-records", handlers.NotificationRecordsRouter)
					r.Route("/organization", handlers.OrganizationRouter)
					r.Route("/organizations", handlers.OrganizationsRouter)
					r.Route("/secrets", handlers.SecretsRouter)
					r.Route("/settings", handlers.SettingsRouter)
					r.Route("/tutorial-progress", handlers.TutorialsRouter)
					r.Route("/user-accounts", handlers.UserAccountsRouter)
				})
			})

			// agent connect and download routes go here (authenticated but with accessKeyId and accessKeySecret)
			r.Group(func(r chiopenapi.Router) {
				r.Use(
					middleware.OTEL(tracers.Agent()),
				)

				r.Route("/", handlers.AgentRouter)
			})
		})
	}
}

func InternalRouter() http.Handler {
	router := chi.NewRouter()
	router.Route("/", handlers.InternalRouter)
	return router
}

func StatusRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return router
}

func ReadyRouter(db *pgxpool.Pool) http.Handler {
	router := chi.NewRouter()
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		var result int
		err := db.QueryRow(r.Context(), "SELECT 1").Scan(&result)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ready":false}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready":true}`))
	})
	return router
}

func FrontendRouter() http.Handler {
	router := chi.NewRouter()
	router.Use(
		chimiddleware.Compress(5, "text/html", "text/css", "text/javascript"),
	)

	router.Handle("/*", handlers.StaticFileHandler(frontend.BrowserFS()))

	return router
}

func WellKnownRouter() http.Handler {
	router := chi.NewRouter()
	if env.WellKnownMicrosoftIdentityAssociation() != nil {
		router.Get("/microsoft-identity-association.json", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(env.WellKnownMicrosoftIdentityAssociation())
		})
	}

	return router
}
