# PR-002 - Environment Domain Model

## User Story

As a Hub administrator, I want to define reusable deployment environments such as Development, Staging, and Production so future releases and lifecycles can use a shared promotion model.

## Scope

PR-002 adds only the Environment domain model. It does not add lifecycle phases, channel rules, release bundles, deployment target assignments, deployment planning, approvals, or retention behavior.

## Feature Flag

The API and UI are guarded by the `environments` experimental feature flag from PR-001.

Example:

```shell
DISTR_EXPERIMENTAL_FEATURE_FLAGS=environments
```

## Data Model

Adds `Environment` with:

- `id`
- `created_at`
- `updated_at`
- `organization_id`
- `name`
- `description`
- `sort_order`
- `is_production`
- `allow_dynamic_targets`
- `retention_policy_id`

Environment names are unique per organization. `sort_order` must be non-negative.

## API

Admin/vendor scoped endpoints:

```http
GET    /api/v1/environments
POST   /api/v1/environments
GET    /api/v1/environments/{environmentId}
PUT    /api/v1/environments/{environmentId}
DELETE /api/v1/environments/{environmentId}
```

Writes require read-write or admin role and are blocked for super admins.

## UI

Adds an admin-only Environments page with:

- loading state;
- error/retry state;
- empty state;
- list/filter table;
- create dialog;
- update dialog;
- delete confirmation.

## Compatibility

Existing deployment target behavior is preserved. PR-002 does not assign deployment targets to environments and does not alter agent protocol or deployment execution.

## Tests

Focused checks added for:

- API request validation;
- internal-to-API mapping;
- handler conversion helpers;
- Angular environment service endpoint behavior.
