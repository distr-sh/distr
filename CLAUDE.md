# CLAUDE.md

Guidance for Claude Code (claude.ai/code) and other agents working in this repository.

## Project

Distr distributes applications to self-managed customers. A control plane (which should be referred to as Distr - no longer as Hub) runs in the cloud, agents run in customer environments, and an OCI-compatible registry serves the artifacts.

## Repository layout

- `cmd/hub/`: Distr itself, a Go backend on chi/v5 serving the REST API on `/api/v1` and the compiled frontend on `/`.
- `cmd/agent/docker/`, `cmd/agent/kubernetes/`: the agents that run Docker Compose and Helm deployments in customer environments and report logs and metrics back.
- `frontend/ui/`: the Angular app (standalone components, TailwindCSS 4, SCSS, Flowbite), built into `internal/frontend/dist/ui/`.
- `sdk/js/`: `@distr-sh/distr-sdk`, a standalone pnpm project. Prefer its high-level `DistrService` over the low-level `Client`, and run its examples against the config in `src/examples/config.ts`.
- `website/`: the docs and marketing site, a standalone pnpm project with a Prettier config of its own.
- `internal/`: the backend. `db/` holds every query, `handlers/` the HTTP handlers, `api/` their request and response structs, `routing/` the routes and middleware, `mapping/` the conversions between `api` and `types`, `types/` the database models, `svc/` the services, `authn/` the authentication providers, `logstore/` the Loki-backed log records, `registry/` the OCI registry, `advisory/` the advisory visibility rules, and `migrations/sql/` the schema.

PostgreSQL is reached through pgx/v5, registry blobs and Loki chunks live in S3-compatible object storage, and deployment logs live in Loki rather than in the database. Timestamp columns are `TIMESTAMP`, never `TIMESTAMPTZ`.

## Commands

Build, test, lint and format through mise. Never invoke `go build`, `go test`, `golangci-lint` or `pnpm` directly.

```sh
mise run build:hub:community   # includes the frontend
mise run build:agent:docker
mise run build:agent:kubernetes
mise run build:sdk             # after every SDK change
mise run build:website
mise run test:go
mise run test:frontend
mise run lint                  # :app, :go, :frontend, :migrations, :website for parts
mise run format                # everything; :app, :go, :frontend, :website for parts
```

Binaries land in `dist/`. Go formatting is configured in `.golangci.yml` and the frontend in `.prettierrc.mjs`; `mise run format:website` also deletes unreferenced images.

## Go Code

- Use `context.Context` for request-scoped values and cancellation, and the `internal/context` helpers to read the logger, database and user from it.
- Do not add new accessors to `internal/context`. They belong in the package that defines the stored type (e.g. `logstore.NewContext`/`logstore.FromContext`), which also avoids import cycles.
- Query through the `internal/db/queryable.Queryable` interface, which covers both `*pgxpool.Pool` and `pgx.Tx`, and `defer rows.Close()` after every query.
- Pass dependencies into HTTP handlers via closure.
- Return API errors through `internal/apierrors` so they carry a status code.
- Log with zap: `logger.Info("message", zap.String("key", value))`.
- Report exceptions with `sentry.GetHubFromContext(ctx).CaptureException(err)`. Use `sentry.CurrentHub()` in a background job, since a job context carries no hub and taking one from it panics.
- Give types in `internal/types` `db:` tags only. Never serialize one into a response and never embed one in an `api` type. `api.OrganizationResponse` and `api.LicenseKeyRevision` do embed one; they are legacy, do not copy them.
- Give every endpoint its own struct in `api/` and put both conversion directions in `internal/mapping`: `XToAPI` for model to response, `XToInternal` for request to model, with `mapping.List(...)` for slices. Never assemble an `api.*` or `types.*` struct field by field in a handler.
- Reference shared string enums (`types.UserRole`, `types.DomainType`, `types.OIDCProvider`) from `api` instead of duplicating them.
- Use `new(value)` to obtain a `*T` from a typed value (e.g. `new(types.UserRoleReadOnly)`). Do not use `util.PtrTo`.
- Use `errors.AsType[E](err)` where the target type is known at the call site, since it needs no pre-declared variable: `if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.UniqueViolation`. Keep `errors.As` where the target is an interface a caller passes in.
- Write 4xx bodies for the end user. The frontend forms display them verbatim (`getFormDisplayedError`), so put anything only a developer can use into the log instead.

## Frontend Code

- Use standalone components (no NgModules), reactive forms, and `inject()` rather than constructor injection for dependencies (`private readonly http = inject(HttpClient)`). `standalone: true` is the default and never needs writing, and neither does `changeDetection: ChangeDetectionStrategy.OnPush`. Set `changeDetection` only to opt a component out with `ChangeDetectionStrategy.Eager`, and drop that opt-out once the component's state is fully signal-based.
- Give services `providedIn: 'root'`.
- Split a component into `component-name.component.ts` and `.html`, plus a `.scss` only when it needs styling beyond utility classes in the template.
- Type API models with the interfaces in `app/types/`, and avoid `undefined` types in favor of the actual type.
- Use [signals](https://angular.dev/guide/signals) for inputs, child views and anywhere else the current Angular version supports them, and convert the non-signal usages you come across in files you edit anyway.
- Use `takeUntilDestroyed` rather than a manual `destroyed$` subject, and [CSS-based animations](https://angular.dev/guide/animations) rather than using the deprecated `@angular/animations` package.
- Self-close components without content: `<fa-icon [icon]="faPlus" />`.
- Take icons from the icon library, never as an SVG path, and keep the name identical in the import, the component and the template (`faServer`, not `serverIcon`). This covers CSS too: never hand-write an inline SVG or a `url("data:image/svg+xml,...")` background, not even to restyle a browser or Flowbite default.
- Leave responsive classes out of modals. They are for desktop only.
- Never bind a template to a method call (`@if (getUsage(x))`, `{{ formatFoo(y) }}`); use a `computed` instead. Every template expression is re-evaluated on each change detection pass, once per instance of the view, so a page with many rows multiplies the work. Anything a template reads indirectly has to be just as cheap: a getter or service method must not decode, parse or re-derive (see the claims cache in `AuthService`).
- Never let a pipe perform I/O. Angular creates one pipe instance per binding, so an `HttpClient` call inside `transform` becomes one request per element, and the browser's HTTP cache does not help because every instance still runs the interceptor chain. Put the request in a `providedIn: 'root'` service that caches by key and shares the in-flight observable with `shareReplay`, and reduce the pipe to a delegate (see `SecureImageCache` and `SecureImagePipe` in `frontend/ui/src/util/secureImage.ts`).
- Call a `ReactiveList`-backed service's `refresh()` before opening a form or modal that has to show the current state, and await it before building the form. Such a service fetches its list once per session, so anything created in another tab, by another user or through the API is missing from it, and a list that arrives later updates the options but no longer moves a selection the form already made.
- Return a `Promise` from a service method that exists for its side effect, `refresh()` being the example, so a caller cannot forget to subscribe. Everything that is a stream (`list()`, the CRUD methods) stays an `Observable`, and a reactive consumer wraps the promise in `from(...)`.
- Never start an HTTP request or other async work inside a `subscribe` callback. Compose it with `switchMap`, or write the handler as an `async` method with `await firstValueFrom(...)`, which is what most event handlers here do.
- Transform text with Tailwind utilities (`capitalize`, `uppercase`, `lowercase`) rather than in TypeScript.
- Pluralize with [NgPlural](https://angular.dev/api/common/NgPlural) instead of a ternary like `count === 1 ? 'day' : 'days'`:

  ```html
  <ng-container [ngPlural]="count()">
    <ng-template ngPluralCase="=1">day</ng-template>
    <ng-template ngPluralCase="other">days</ng-template>
  </ng-container>
  ```

- Reuse the shared `distr-*` classes from `frontend/ui/src/styles/theme.scss` (`distr-input`, `distr-checkbox`, `distr-radio`, `distr-label`, …) instead of repeating their utility chains inline, and add one there when an element's styling repeats across the app. Append only what is specific to the element (e.g. `indeterminate:bg-[length:65%_65%]` on a `distr-checkbox`), and express a state Tailwind has a variant for (`disabled:`, `read-only:`) inside the shared class so every instance behaves the same. Define a variant class such as `distr-input-raised` below the class it overrides, since `@layer components` resolves by source order. Tailwind scans this file, so describe a class rather than pasting a full `class="..."` attribute into it.
- Check how a shared control is already used before inventing a pattern for it. The indeterminate "select all" checkbox, for example, needs nothing beyond `distr-checkbox`.
- Put `app-page` in routed components only, exactly once per rendered page, never in a component embedded in another page. When such a component is also routed on its own, give that route a thin wrapper that adds the `app-page` (e.g. `CustomerLicenseDetailPageComponent`, `ApplicationsPageComponent`).
- Put styling a component always needs on the component itself via `host: {class: '…'}` (e.g. `app-search-bar`, `app-editor`), and leave only what varies per call site in the template.
- Keep styling used by a single component in its own `.scss` rather than in `theme.scss`, and start that file with `@reference '<path>/styles/tailwind.css'`, or `@apply` fails the build with `Cannot apply unknown utility class`. Tailwind compiles each component stylesheet on its own, and that entry point holds the `@theme` tokens and the `dark` variant, so keep it plain CSS: `@reference` cannot read Sass. Prefer the `.scss` file over an inline `styles` block, since Tailwind scans `.ts` files and emits utilities found there into the global stylesheet as well.
- Never map Tailwind utility chains in the component class to pick a variant at runtime. Put the variants into the stylesheet.
- Use a shared badge class for every badge: `distr-status-badge` for a state, with the colors including a `border-*` from a color helper next to the feature, `distr-tag-badge` for a free-form label, and `distr-deployment-type-badge` or `distr-artifact-tag` for those two. Do not write a new pill inline.
- Use `app-badge-select` where the state a badge shows can also be set, rather than a row of buttons or a separate `<select>`. It emits the picked value instead of writing it, so the caller sends the request and passes back what the server returned.
- Put `distr-dropdown-panel` on the root element of every dropdown overlay: a wrapping `<div>` when the panel holds more than the list, the `<ul>` itself when it does not. Give the entries `distr-dropdown-item`, with `distr-dropdown-item-danger` for a destructive action and `distr-dropdown-item-warning` for one that needs caution, and put `distr-dropdown-divider` on an empty `<li>` for a section break. The item class goes on the `<button>` or `<a>`, never on the enclosing `<li>`, so the hover area and the click area match. Do not write a dropdown row inline and do not reintroduce the `divide-y` separators the panel replaced. Dropdowns whose entries are checkboxes are the exception: their rows stay bordered and have no hover.

## Database Access

Route every database access through `internal/db/`. Never write raw SQL in a handler or service; add the query to the matching file in `internal/db/` instead.

Use `now()` for the current time, never `current_timestamp`, in `internal/db/` and in the migrations in `internal/migrations/sql/`, including column defaults.

```go
err := db.BeginFunc(ctx, func(tx pgx.Tx) error {
    // Do queries with tx
    return nil
})
```

### Batch Inserts

Insert multiple rows with `pgx.CopyFrom` and `pgx.CopyFromSlice`, never with `INSERT` in a loop.

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

### Enum Types

Model a closed set of values as a Postgres enum type, not as a `TEXT` column with a `CHECK (col IN (...))` constraint. Save `CHECK` for cross-column invariants such as `(type = 'docker') = (scope IS NULL)`.

Register a new enum type and its array type, prefixed with `_`, in the `AfterConnect` type list in `internal/svc/db_pool.go`, e.g. `CUSTOM_DOMAIN_TYPE` and `_CUSTOM_DOMAIN_TYPE`; without it pgx cannot encode Go values into the enum's OID. Cast query parameters to the enum type rather than to `TEXT`, since Postgres does not coerce `text` to an enum: `unnest(@domainTypes::CUSTOM_DOMAIN_TYPE[])`. Pass the Go string type itself (`[]types.DomainType`), not `[]string`.

### Encrypted Columns

Read the doc comments of `internal/dbcrypto` and `internal/db/encryption.go` before you touch an encrypted column; they explain the stored format and what a value is bound to.

- Declare every encrypted column once in `internal/db/encryption.go`, as a package-level variable that `EncryptedColumns` also lists, and reference that variable from the queries that read and write it. Never name such a column as a loose string at a call site, and never leave one out of `EncryptedColumns`.
- Type the field in `internal/types` as `dbcrypto.String`, `*dbcrypto.String` or `dbcrypto.Bytes`, never as `string` or `[]byte`.
- Read through `column.Output(alias)`, or `column.Value(alias)` where a column alias is not allowed, and use `column.IsSetValue(alias)` for the boolean an API exposes in place of the secret itself.
- Write only the `_enc` column, from `column.Encrypt`, `EncryptPtr` or `EncryptBytes`, and set the plaintext column to `NULL` in the same statement.
- Declare the scope as every column of the same row that an authorization check reads, most specific first, e.g. `encrypted("Secret", "value", "customer_organization_id", "organization_id")`, and pass `dbcrypto.ScopeOf` for a nullable one.
- A statement that writes a scope column has to write the encrypted columns of that row in the same statement, and one that does not has to require the scope it sealed for in its `WHERE` (see `UpdateSecret` and `UpdateArtifactUpstream`).
- Treat renaming a table or a column, or changing what a column is scoped to, as a data migration: it invalidates every value in that column until `maintenance encrypt-database` has rewritten them.
- Call `dbcrypto.Init(env.DatabaseEncryptionKey())` in the `PreRun` of every command that touches an encrypted column, and never make `dbcrypto` read `env` itself.
- Do not encrypt a column that a query looks up by value. Narrow the row down by its id and compare in Go with `subtle.ConstantTimeCompare` (see `db.GetSupportBundleByBundleSecret`).

### Read-only Database

`DATABASE_READONLY_URL` optionally configures a replica, which `ContextInjectorMiddleware` injects with `WithReadonlyDB`. Apply `middleware.UseReadonlyDB` to a route to serve it from there; the middleware swaps the context's active db so every `db.*` call in the handler follows, and does nothing when no replica is configured.

- Apply it only to routes that run read-only queries exclusively.
- Prefer logs, analytics, dashboards, metrics and status timeseries. Keep it away from anything the frontend refetches after an update, and from read-after-write workloads in general, since the replica may lag. The OCI registry always uses the primary: container clients rely on immediate consistency for push then pull, multi-arch index push and signing.
- Place it after the authentication and authorization middleware so those lookups keep hitting the primary. Applying it at the router mount (`r.With(middleware.UseReadonlyDB).Route(...)`) is fine when the whole router is read-only; otherwise wrap only the read routes in a group.

## Scheduled Jobs

A job has to be runnable from outside the Distr process, since a high-availability installation would otherwise run it once per replica. Register it in `internal/svc/jobs_scheduler.go` behind its own `*_CRON` env var that defaults to unscheduled, give it a subcommand (`cleanup` for pruning, `maintenance` for everything else), and add a `cronJobs` entry to `deploy/charts/distr/values.yaml` that calls it. Never make behavior outside the job depend on whether its cron is scheduled: in the chart it never is, because the CronJob runs it.

Give a one-time migration such as `maintenance encrypt-database` the subcommand and nothing else: no `*_CRON` env var, no `cronJobs` entry and no Helm hook. Document the command on the website instead, and let Distr log on startup that there is work left.

## Subscription Gating

Never gate a feature by listing the subscription types allowed to use it. Such an allowlist has to be touched again for every new plan, and the plan silently loses the feature when someone forgets. Express gating as a denylist of the lower plans, so a new plan gets access by default:

- Go: `types.NonProSubscriptionTypes` with `SubscriptionType.IsPro()`, `middleware.ForbidSubscriptionTypes(...)` or the ready-made `middleware.ProFeature`.
- Frontend: `isProSubscription()` / `isPayingSubscription()` from `app/types/subscription.ts`, or `NON_PRO_SUBSCRIPTION_TYPES` / `NON_PAYING_SUBSCRIPTION_TYPES` when a list is needed.

Plan-specific billing UI (checkout, plan comparison) and upsell banners for one plan are the exceptions, since they are tied to concrete plans by nature.

Keep the two sources of organization features (`types.Feature`) apart. Only the plan-managed ones, granted by `types.FeaturesForSubscriptionType` and collected in `types.PlanManagedFeatures`, may be revoked when an organization loses its plan. The rest is granted out of band (`vendor_billing` by staff, `pre_post_scripts` and `artifact_version_mutable` by an organization admin in the settings) and has to survive plan changes and edition reconciliation. Remove `types.PlanManagedFeatures` from the `features` array to revoke a plan; never overwrite the whole array.

## Sending Mail

Never take the mailer straight from the context. An organization can configure its own SMTP server (`CustomEmailConfiguration`), which overrides the instance mailer built from the `MAILER_*` env vars, so resolve the transport with `custommail.MailerForOrganization(ctx, orgID)` and the sender address with `custommail.FromAddressOrDefault(ctx, orgID, branding)`. Both fall back to the instance defaults when the organization has no enabled configuration. Neither falls back when sending through a configured server fails, because the instance mailer would then send from a domain that server's sender address does not belong to, which fails SPF, DKIM and DMARC and hides the misconfiguration.

Pass the organization explicitly rather than reading it from the authentication. Background jobs carry no authentication (`internal/jobs/runner.go`), and notification mail is sent from exactly there.

## Input Trimming

Never trim a single request field by hand. `validation.TrimStrings` in `handlers.JsonBody` and the frontend's `trimInterceptor` trim request body strings generically; where surrounding whitespace matters, opt out with `trim:"-"` and `skipTrim()`.

## API Routes

Define routes in `internal/routing/`, grouped by what they authenticate with: public, user (JWT), admin, agent token and the OCI registry's own scheme.

Keep the OpenAPI spec valid, which the `chiopenapi` router generates from the route definitions. Declare path parameters, query parameters and request bodies with `option.Request()` and a struct using `path:`, `query:` and `json:` tags, composing path param structs with body request structs by embedding as the existing routes do. An endpoint with neither parameters nor a body needs no `option.Request()`.

## Generated URLs

Never hard-wire `https://` into a URL built for this instance. A hard-wired scheme breaks every locally running instance, and for the OIDC callback URL it produces a URL that disagrees with the `redirect_uri` the login sends. Take the scheme from `env.HostScheme()`, which reads `DISTR_HOST` (https unless it says http) and returns the `env.URLScheme` enum (`env.SchemeHTTP` / `env.SchemeHTTPS`); compare a parsed URL's scheme against that enum rather than against a bare `"https"` string.

- On the host of the current request, use `handlerutil.GetRequestSchemeAndHost(r)`. It keeps the request's host, so a request on a custom domain stays there, and takes the scheme from the configuration rather than the request, which arrives as plain http behind a TLS-terminating proxy.
- On another host, use `env.HostScheme()` directly: an organization's custom domain in `customdomains.withScheme`, the login forwarding target, and the OIDC callback URL an administrator has to register (`oidc.CustomCallbackURL`).
- In the frontend, use the protocol of the current page.

Build the host through the `internal/customdomains` resolvers, never from `db.GetCustomDomains` directly, since only the resolvers drop unverified domains. Do not add that filter anywhere else. Listing a caller's domains and resolving the host of an incoming request deliberately accept unverified ones.

## Comments

Write as few comments as possible. A comment has to say something the code cannot; anything else is noise that goes stale and has to be reviewed forever.

Do not write a comment that:

- Restates the code or the name below it, including a doc comment on a self-explanatory type, field, function or env getter. A getter named after the value it returns needs no comment saying that it returns that value.
- Explains the change you are making, why it is correct, or what was there before. That belongs in the commit message or the pull request description.
- Narrates a step of an obvious sequence (`// send the request`, `// parse the response`).
- Explains a styling or layout choice in a template, or a language detail a reader of that language already knows.
- Repeats the documentation, a rule in this file, or a linked ticket.

Do write a comment that records something a reader cannot see:

- A constraint imposed from outside the code, such as a requirement of a third-party API, a browser or protocol quirk, or a database limitation the code works around.
- Why a non-obvious approach was chosen, when the obvious one is wrong or breaks something.
- A deliberate invariant that a future change would silently break.

## Tests

Only write a test that could fail for a real reason. Every test is code that has to be maintained, and one that restates the implementation costs maintenance without ever catching a bug.

- Assert with [Gomega](https://onsi.github.io/gomega/) in Go tests.
- Do not test guard clauses, getters, plain mappings, a single `if` branch, or that a value passed in comes back out.
- Do not write a test whose assertion is trivially true because the dependency it needs is not configured in tests.
- Do test what is hard to get right and expensive to get wrong: wire formats sent to third parties, fail-closed security behavior, parsing, permission and subscription gating, and non-trivial query or business logic.
- Prefer a few focused tests over an exhaustive matrix of near-duplicates.

## General rules

- Keep this file current, and add a rule for it whenever a user asks you to do something differently than you did.
- This file holds instructions and conventions for the agent, not technical documentation. Add a rule that changes what an agent does; never a description of how a feature, endpoint or subsystem works. That belongs in the code, in a doc comment, or on the website.
  - Write every rule as an instruction: what to do, what not to do, and where. A rule needs no paragraph explaining the design it protects, no threat model and no history of what was there before. Where an agent has to understand the mechanism to follow the rule, point it at the doc comment or the docs page that explains it rather than restating them here, since a copy here goes stale without anything failing.
  - When you add a rule, check whether the surrounding section can lose prose in exchange. This file is read in full on every request, so its length is a cost paid by every task.
- Fix the code you read that breaks these rules, and the typos and spelling mistakes you come across.
- Update `website/src/content/docs/docs/self-hosting/configuration.mdx` in the same change whenever you add, remove or change an environment variable in `internal/env/env.go`, including its default, whether it is required, and the values it accepts.
- Use the GitHub CLI (`gh`) rather than the web interface to fetch data from GitHub.
- Write shell for anything from a one-off command to a checked-in script (like `hack/validate-migrations.sh`), and Node once a task outgrows shell (like `hack/agent-changelog.mjs`). Avoid Python and never use Perl (e.g. `perl -pi -e`). Edit files directly rather than piping them through a stream editor.

## Code Review Instructions

Follow @.github/copilot-code-review-instructions.md when performing code reviews.
