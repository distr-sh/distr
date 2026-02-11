# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Distr is an open-source software distribution platform that enables companies to distribute applications to self-managed customers.
It provides centralized management of deployments, artifacts, agents, licenses, and includes an OCI-compatible container registry.
The platform consists of a control plane (Hub) running in the cloud, agents that run in customer environments, and an MCP server for AI integrations.

## Architecture

### High-Level Components

1. **Distr Hub** (`cmd/hub/`): The main control plane server
   - Go backend with chi router
   - Angular 20 frontend (TypeScript, TailwindCSS 4)
   - REST API at `/api/v1`
   - Serves the compiled frontend on root path

2. **Agents** (`cmd/agent/`):
   - `docker/`: Docker agent for managing Docker Compose deployments
   - `kubernetes/`: Kubernetes agent for managing Helm deployments
   - Agents connect to Hub, collect logs/metrics, execute deployments

3. **MCP Server** (`cmd/mcp/`): Model Context Protocol server for AI integrations

4. **SDK** (`sdk/js/`): JavaScript/TypeScript SDK for interacting with Distr API

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

- Always build the SDK with `pnpm build` after making changes
- Use pnpm (not npm) for all package management
- Use `DistrService` for high-level operations (preferred)
- Use `Client` for direct API access when needed
- Example files use a config from `src/examples/config.ts`

### Backend Architecture (Go)

- **Database**: PostgreSQL accessed via pgx/v5 with connection pooling
- **Router**: chi/v5 with middleware-based architecture
- **Authentication**: JWT-based with support for OIDC, API keys, and agent tokens
- **OCI Registry**: Adapted from google/go-containerregistry for serving Docker images, Helm charts, and other artifacts
- **Storage**: S3-compatible object storage (MinIO for dev) for registry blobs
- **Migrations**: SQL migrations in `internal/migrations/sql/` managed by golang-migrate
- **Database queries**: All database interactions are in `internal/db/` with transaction support

Key internal packages:

- `internal/handlers/`: HTTP request handlers
- `internal/routing/`: Route configuration and middleware setup
- `internal/authn/`: Authentication providers (JWT, API keys, agent tokens)
- `internal/db/`: Database queries and models
- `internal/registry/`: OCI registry implementation
- `internal/middleware/`: HTTP middleware (logging, auth, Sentry, etc.)
- `internal/svc/`: Business logic services
- `internal/mapping/`: Mapping logic for data transformations between DTOs and domain models

### Frontend Architecture (Angular)

- **Framework**: Angular 21 with standalone components
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
- `licenses`: License keys for artifact access
- `deployment_log_records`: Logs from deployments

## Common Commands

### Building

```bash
# Build hub (includes frontend build)
mise run build:hub:community        # Community edition

# Build agents
mise run build:agent:docker
mise run build:agent:kubernetes

# Build MCP server
mise run build:mcp
```

Binaries are output to `dist/`.

### Linting and Formatting

```bash
# Auto-fix linting issues
mise run format              # All
mise run format:go           # Go only
mise run format:frontend     # Frontend only
```

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
- Use structured logging with zap: `logger.Info("message", zap.String("key", value))`
- Send exceptions to sentry with: `sentry.GetHubFromContext(ctx).CaptureException(err)`
- When performing data transformations between DTOs and domain models, use `mapping.List(...)` inside the `internal/mapping` package

### Frontend Code

- Use standalone components (no NgModules) - This is the default so `standalone: true` is not needed
- Services are singleton by default (`providedIn: 'root'`)
- Use Angular's HttpClient for API calls, injected via constructor
- Component file structure: `component-name.component.ts`, `component-name.component.html` (no need for scss files)
- Use TypeScript interfaces from `app/types/` for API models
- Use reactive forms for all form handling
- Use as little `undefined` types as possible, always use the actual type
- Don't use any svg path icons, always look for a matching icon in the icon library used. These icons should always be the same in the import, the component and template e.g. `faServer` and not `serverIcon`.
- Use [Angular Signals](https://angular.dev/guide/signals) for inputs, child views and everywhere where the current Angular version supports signals.
  If you find usages of non signal usages for inputs, child views etc. change them to signals in the files you would edit anyway.
- Don't use any responsive design classes in modals. They should always be optimized for the none mobile use case.
- Use Angular's `takeUntilDestroyed` instead of a manual `destroyed$` subject.

### Database Access

All database access should go through `internal/db/` functions. Never write raw SQL in handlers or services. If you need a new query, add it to the appropriate file in `internal/db/`.

Transaction pattern:

```go
err := db.BeginFunc(ctx, func(tx pgx.Tx) error {
    // Do queries with tx
    return nil
})
```

### API Routes

API routes are defined in `internal/routing/`. Routes are grouped by authentication requirements:

- Public routes (no auth)
- User routes (JWT auth required)
- Admin routes (admin user required)
- Agent routes (agent token auth)
- Registry routes (special OCI auth)

## General rules

- Always ensure this file is up-to-date.
- Don't write any unnecessary comments that just explain the functionality below, if there is nothing special about it.
- If a user requests you to do something differently, add the difference to a new rule / convention in this file
- If you read code that doesn't follow these rules, please fix it.
- If you see any typos, or spelling mistakes, please fix them.
- If you fetch data from GitHub always use the GitHub cli (`gh`) instead of the web interface.
- When you resolve merge conflicts (whether during a merge or rebase), always ensure that the conflict resolutions are committed before continuing, or at least prompt the user to commit them, so that unrelated new changes are not unintentionally included in that commit.
