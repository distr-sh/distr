# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Distr is an open-source software distribution platform that enables companies to distribute applications to self-managed customers.
It provides centralized management of deployments, artifacts, agents, licenses, and includes an OCI-compatible container registry.
The platform consists of a control plane (Hub) running in the cloud and agents that run in customer environments.

## Architecture

### High-Level Components

1. **Distr Hub** (`cmd/hub/`): The main control plane server
   - Go backend with chi router
   - Angular frontend (TypeScript, TailwindCSS 4)
   - REST API at `/api/v1`
   - Serves the compiled frontend on root path

2. **Agents** (`cmd/agent/`):
   - `docker/`: Docker agent for managing Docker Compose deployments
   - `kubernetes/`: Kubernetes agent for managing Helm deployments
   - Agents connect to Hub, collect logs/metrics, execute deployments

3. **SDK** (`sdk/js/`): JavaScript/TypeScript SDK for interacting with Distr API

### SDK Architecture (TypeScript)

The SDK is a standalone subproject in `sdk/js/` with its own package.json, dependencies, and build process.

- **Location**: `sdk/js/`
- **Package**: `@distr-sh/distr-sdk`
- **Package Manager**: pnpm
- **Build**: `pnpm build` (compiles TypeScript to `dist/`)
- **Test**: `pnpm test:examples` (runs example test client)
- **Examples**: `sdk/js/src/examples/` contains usage examples
- **Main classes**:
  - `Client`: Low-level API client (in `src/client/client.ts`)
  - `DistrService`: High-level service with convenience methods (in `src/client/service.ts`)

When working with the SDK:

- Always build the SDK with `mise build:sdk` after making changes
- Use pnpm (not npm) for all package management
- Use `DistrService` for high-level operations (preferred)
- Use `Client` for direct API access when needed
- Example files use a config from `src/examples/config.ts`

### Backend Architecture (Go)

- **Database**: PostgreSQL accessed via pgx/v5 with connection pooling
- **Router**: chi/v5 with middleware-based architecture
- **Authentication**: JWT-based with support for OIDC, API keys, and agent tokens
- **OCI Registry**: Adapted from google/go-containerregistry for serving Docker images, Helm charts, and other artifacts
- **Storage**: S3-compatible object storage (rustfs for dev) for registry blobs and Loki log chunks
- **Log storage**: Deployment and deployment target log records are stored in Grafana Loki (not PostgreSQL), accessed through the `internal/logstore` package (`LogStore` interface, Loki implementation, in-memory fake for tests). The log record types (`logstore.DeploymentLogRecord`, `logstore.DeploymentTargetLogRecord`) live in this package, not in `internal/types`, since they are not database entities. The org ID is passed explicitly to every store method and maps to the Loki tenant (`X-Scope-OrgID`). Log retention is time-based and managed by the Loki config (shipped default: 30 days); read queries are limited to a subscription-dependent query window (`subscription.GetLogQueryWindow`: 24 hours for Community/Starter, 7 days otherwise). Log exports (deployment logs, deployment target logs and deployment status) honor the `before`/`after`/`filter` query parameters of the log viewer and are hard-capped at `subscription.MaxLogExportRows` (1,000,000 lines) for every subscription type; truncated exports end with a notice in the downloaded file
- **Migrations**: SQL migrations in `internal/migrations/sql/` managed by golang-migrate
- **Database queries**: All database interactions are in `internal/db/` with transaction support

Key internal packages:

- `internal/handlers/`: HTTP request handlers
- `internal/routing/`: Route configuration and middleware setup
- `internal/authn/`: Authentication providers (JWT, API keys, agent tokens)
- `internal/db/`: Database queries and models
- `internal/logstore/`: Log record storage (Loki-backed, with in-memory fake for tests)
- `internal/registry/`: OCI registry implementation
- `internal/middleware/`: HTTP middleware (logging, auth, Sentry, etc.)
- `internal/svc/`: Business logic services
- `internal/mapping/`: Mapping logic for data transformations between DTOs and domain models
- `api/`: All request structs used by HTTP handlers should be in the api package and not in the handler package

### Frontend Architecture (Angular)

- **Framework**: Angular with standalone components
- **Styling**: TailwindCSS 4, SCSS, Flowbite components
- **Routing**: Angular Router with lazy-loaded routes
- **State**: Service-based state management
- **Forms**: Reactive forms with Angular Forms
- **Key directories**:
  - `frontend/ui/src/app/`: All application components
  - `frontend/ui/src/app/services/`: Data services and API clients
  - `frontend/ui/src/app/components/`: Reusable UI components
  - `frontend/ui/src/buildconfig/`: Build-time configuration injected by Go

The frontend is built into `internal/frontend/dist/ui/` and served by the Go backend.

### Database Schema

The database schema is managed through SQL migrations in `internal/migrations/sql/`. Key tables include:

- `user_accounts`: User authentication and profiles
- `organizations`: Multi-tenant organizations
- `deployments`: Application deployments
- `deployment_targets`: Customer environments (agents)
- `artifacts`: Software artifacts (Docker images, Helm charts)
- `applications`: Artifact collections
- `licensekey`: License keys that vendors can generate for its customers
- `application_entitlements` & `artifact_entitlements`: Access entitlements for applications and artifacts

This database stores timestamps as `TIMESTAMP` (without time zone), not `TIMESTAMPTZ`.

## Common Commands

### Building

````bash
# Build hub (includes frontend build)
mise run build:hub:community        # Community edition

# Build agents
mise run build:agent:docker
mise run build:agent:kubernetes

Binaries are output to `dist/`.

### Linting and Formatting

```bash
# Auto-fix linting issues
mise run format              # All
mise run format:go           # Go only
mise run format:frontend     # Frontend only
````

Go linting uses golangci-lint with config in `.golangci.yml`. Frontend uses Prettier with config in `.prettierrc.mjs`.

## Code Patterns and Conventions

### Go Code

- Use `context.Context` for request-scoped values and cancellation
- Database queries return `pgx.Rows` or use `pgx.QueryRow` for single rows
- Always use `defer rows.Close()` after querying
- Use `internal/db/queryable.Queryable` interface for queries (supports both `*pgxpool.Pool` and `pgx.Tx`)
- HTTP handlers receive dependencies via closure (database pool, logger, etc.)
- Error handling uses `internal/apierrors` for API errors with proper status codes
- Use `internal/context` helpers to retrieve logger, database, user from context
- Do not add new context accessors to `internal/context`. Following idiomatic Go, they belong in the package that defines the stored type (e.g. `logstore.NewContext`/`logstore.FromContext`), which also avoids import cycles
- Use structured logging with zap: `logger.Info("message", zap.String("key", value))`
- Send exceptions to sentry with: `sentry.GetHubFromContext(ctx).CaptureException(err)`
- When performing data transformations between DTOs and domain models, use `mapping.List(...)` inside the `internal/mapping` package
- Always use [Gomega](https://onsi.github.io/gomega/) for test assertions in Go tests
- Do not use `util.PtrTo`. Use `new(value)` to obtain a `*T` from a typed value (e.g. `new(types.UserRoleReadOnly)`).

### Frontend Code

- Always use self-closing tags for Angular components when they have no content (e.g. `<fa-icon [icon]="faPlus" />` instead of `<fa-icon [icon]="faPlus"></fa-icon>`)
- Use standalone components (no NgModules) - This is the default so `standalone: true` is not needed
- Services are singleton by default (`providedIn: 'root'`)
- Use Angular's `inject()` function for dependency injection (e.g. `private readonly http = inject(HttpClient)`). Do not use constructor injection.
- Component file structure: `component-name.component.ts`, `component-name.component.html` (no need for scss files)
- Use TypeScript interfaces from `app/types/` for API models
- Use reactive forms for all form handling
- Use as little `undefined` types as possible, always use the actual type
- Don't use any svg path icons, always look for a matching icon in the icon library used. These icons should always be the same in the import, the component and template e.g. `faServer` and not `serverIcon`.
- Use [Angular Signals](https://angular.dev/guide/signals) for inputs, child views and everywhere where the current Angular version supports signals.
  If you find usages of non signal usages for inputs, child views etc. change them to signals in the files you would edit anyway.
- Don't use any responsive design classes in modals. They should always be optimized for the none mobile use case.
- Use Angular's `takeUntilDestroyed` instead of a manual `destroyed$` subject.
- Use [Angular Signal Based Animations](https://angular.dev/guide/animations) instead of legacy animations defined in the component.
- Use Tailwind CSS utility classes for text transformations (e.g. `capitalize`, `uppercase`, `lowercase`) instead of TypeScript string manipulation when possible.
- Use [NgPlural](https://angular.dev/api/common/NgPlural) with `ngPluralCase` for pluralized text in templates instead of ternaries like `count === 1 ? 'day' : 'days'`:

  ```html
  <ng-container [ngPlural]="count()">
    <ng-template ngPluralCase="=1">day</ng-template>
    <ng-template ngPluralCase="other">days</ng-template>
  </ng-container>
  ```

- Reuse the shared global `distr-*` component classes defined in `frontend/ui/src/styles/theme.scss` (e.g. `distr-checkbox`, `distr-radio`, `distr-label`) instead of repeating their Tailwind utility chains inline, and add a new one there when an element's styling is repeated across the app. Append only element-specific extra utilities when needed (e.g. `class="distr-checkbox indeterminate:bg-[length:65%_65%]"`).

### Database Access

All database access should go through `internal/db/` functions. Never write raw SQL in handlers or services. If you need a new query, add it to the appropriate file in `internal/db/`.

Transaction pattern:

```go
err := db.BeginFunc(ctx, func(tx pgx.Tx) error {
    // Do queries with tx
    return nil
})
```

#### Enum Types

Model a closed set of values as a Postgres enum type, not as a `TEXT` column with a `CHECK (col IN (...))` constraint. `CHECK` constraints are for cross-column invariants (e.g. `(type = 'docker') = (scope IS NULL)`).

When you add a Postgres enum type, register it (and its array type, prefixed with `_`) in the `AfterConnect` type list in `internal/svc/db_pool.go`, e.g. `CUSTOM_DOMAIN_TYPE` and `_CUSTOM_DOMAIN_TYPE`. Without it pgx cannot encode Go values into the enum's OID. Cast query parameters to the enum type, never to `TEXT`: `unnest(@domainTypes::CUSTOM_DOMAIN_TYPE[])`, since Postgres does not implicitly coerce `text` to an enum. Pass the Go string type itself (`[]types.DomainType`), not `[]string`.

#### Read-only Database

An optional read-only database (e.g. a replica) can be configured via `DATABASE_READONLY_URL` (and `DATABASE_READONLY_MAX_CONNS`). When unset, no read-only pool is created and everything uses the primary. When set, it is injected into the request context by `ContextInjectorMiddleware` via `WithReadonlyDB` (the primary is always injected via `WithDb`).

To serve an endpoint from the read-only db, apply the `middleware.UseReadonlyDB` middleware to its route. It swaps the context's active db to the read-only pool so all `db.*` calls in the handler use it, and is a **noop** when no read-only db is configured. Rules:

- Only use it for routes that perform **exclusively read-only** queries.
- Never use it for routes that are part of an update-and-refetch loop in the frontend (the read-only db may lag behind the primary). Good candidates are logs, analytics, dashboards, metrics, and status timeseries.
- Place it **after** authentication/authorization middleware so those lookups keep hitting the primary. Applying it at the router mount (`r.With(middleware.UseReadonlyDB).Route(...)`) is fine when the whole router is read-only; otherwise wrap only the relevant read routes in a group.
- Do not use it for read-after-write workloads. In particular, the OCI registry always uses the primary: container clients rely on immediate consistency (push then pull/HEAD, multi-arch index push, signing), which a lagging replica would break.

### Batch Inserts

Use `pgx.CopyFrom` with `pgx.CopyFromSlice` for inserting multiple rows. Never use individual `INSERT` statements in a loop.

```go
_, err := db.CopyFrom(
    ctx,
    pgx.Identifier{"tablename"},
    []string{"col1", "col2"},
    pgx.CopyFromSlice(len(items), func(i int) ([]any, error) {
        return []any{items[i].Col1, items[i].Col2}, nil
    }),
)
```

### Subscription Gating

Never gate a feature by listing the subscription types that are allowed to use it. Every such allowlist has to be touched again whenever a new plan is introduced, and the plan silently loses the feature if it is forgotten. Always express gating as a denylist of the lower plans instead, so a new plan gets access by default:

- Go: use `types.NonProSubscriptionTypes` with `SubscriptionType.IsPro()`, `middleware.ForbidSubscriptionTypes(...)` or the ready-made `middleware.ProFeature`.
- Frontend: use `isProSubscription()` / `isPayingSubscription()` from `app/types/subscription.ts`, or `NON_PRO_SUBSCRIPTION_TYPES` / `NON_PAYING_SUBSCRIPTION_TYPES` when a list is needed.

The only exceptions are plan-specific billing UI (checkout, plan comparison) and upsell banners for one particular plan, which are inherently tied to concrete plans.

Organization features (`types.Feature`) come from two sources and must not be mixed up. Plan-managed features are granted by `types.FeaturesForSubscriptionType` and collected in `types.PlanManagedFeatures`; they are the only ones that may be revoked when an organization loses its plan. Everything else is granted out of band — `vendor_billing` by staff, `pre_post_scripts` and `artifact_version_mutable` by an organization admin in the settings — and must survive plan changes and edition reconciliation. Never overwrite the whole `features` array to revoke a plan; remove `types.PlanManagedFeatures` from it.

### API Routes

API routes are defined in `internal/routing/`. Routes are grouped by authentication requirements:

- Public routes (no auth)
- User routes (JWT auth required)
- Admin routes (admin user required)
- Agent routes (agent token auth)
- Registry routes (special OCI auth)

When adding new routes, ensure the OpenAPI spec remains valid. The `chiopenapi` router generates the spec from route definitions. Endpoints that have path parameters, query parameters, or a request body must declare them via `option.Request()` with a struct using the appropriate tags (`path:`, `query:`, `json:`). Endpoints without any parameters or body do not need `option.Request()`. Follow the existing pattern of composing path param structs with body request structs via embedding.

### Host-resolved Bootstrap Configuration

`GET /api/public/v1/portal` (`internal/handlers/portal.go`) is the single endpoint the unauthenticated pages boot from: it resolves the request Host to an organization and returns its portal branding plus the login methods available on that host. Anything the login/register pages need before a user exists belongs here, not in a new endpoint — on the frontend it is owned by `PortalService`, which requests it once and replays it.

`resolvePortalHost` distinguishes three host sources. Self-service `CustomDomain` rows and legacy `OrganizationBranding.app_domain` values are **not** interchangeable: both drop Distr's own branding, but the instance-scoped OIDC providers stay available on the legacy domains, since they predate self-service domains and their users would otherwise be locked out. The response is cached per Host (`Vary: Host`, `max-age=60`), so it must not carry anything user- or organization-specific.

### Organization-configured OIDC Providers

A `CustomOIDCConfiguration` is an organization's own identity provider, hanging off the `CustomDomain` it is offered on (`custom_domain_id`). The domain decides both where the provider is usable and who may configure it, which is what lets the same table serve a vendor's app domain today and a customer portal domain later without a schema change.

- **Never cache providers.** `oidc.ProviderForConfiguration` builds one per request from the configuration row, so an edit takes effect immediately. Discovery and JWKS are cached by `go-oidc` within a flow, which is where caching belongs.
- **Store the issuer the provider states for itself**, taken from its discovery document (`oidc.Discover`), never the string an administrator typed. Auth0 states a trailing slash and Entra ID a tenant GUID, and identities are keyed by `(configuration, issuer, subject)`.
- Issuer URLs are administrator-controlled input that the Hub fetches, so anything that resolves one must go through `oidc.ParseIssuerURL` and the SSRF-guarded client in `internal/oidc/discovery.go`.
- **Account exclusivity is a security invariant, not a convenience:** a custom provider may only authenticate an account whose memberships are exclusively in that provider's organization. Without it, an organization can point a provider at an account that also belongs to somebody else's organization and reach their data through a personal access token, which no session-scoping claim prevents. Identity resolution therefore only ever matches within the provider's organization, and `internal/handlers/account_exclusivity.go` guards organization creation and invites. Any new way to gain a membership or an organization has to call those guards.

## General rules

- Always ensure this file is up-to-date.
- Always build, test, lint and format through mise tasks (`mise run build:hub:community`, `mise run test:go`, `mise run test:frontend`, `mise run lint`, `mise run format`). Never invoke `go build`, `go test`, `golangci-lint` or `pnpm` directly.
- When you add, remove, or change an environment variable in `internal/env/env.go` (name, default, required/optional status, or accepted values), update the configuration reference page at `website/src/content/docs/docs/self-hosting/configuration.mdx` in the same change so it stays complete and accurate.
- Don't write any unnecessary comments that just explain the functionality below, if there is nothing special about it. The default for new code is no comment at all: no doc comments on Go functions, types or struct fields that only restate their name, no JSDoc on Angular components and services, no comments in SQL migrations, and no explanations of the design rationale of a change (that belongs in the commit message or the PR description). Write a comment only for a constraint or trap that a reader cannot see in the code itself, and keep it to one or two lines. Design rationale that is worth keeping goes into this file or the docs under `website/`, not into the source.
- If a user requests you to do something differently, add the difference to a new rule / convention in this file
- If you read code that doesn't follow these rules, please fix it.
- If you see any typos, or spelling mistakes, please fix them.
- If you fetch data from GitHub always use the GitHub cli (`gh`) instead of the web interface.
- When you resolve merge conflicts (whether during a merge or rebase), always ensure that the conflict resolutions are committed before continuing, or at least prompt the user to commit them, so that unrelated new changes are not unintentionally included in that commit.

## Code Review Instructions

Follow @.github/copilot-code-review-instructions.md when performing code reviews.
