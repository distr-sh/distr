# ADR-0001: Experimental Feature Flags

## Status

Accepted

## Context

The community roadmap introduces multiple large deployment-management capabilities over many pull requests. Those capabilities must remain hidden until their milestone is complete, while existing Distr deployments continue to work without database or agent changes.

Existing organization features are persisted entitlements used for licensing, billing, and tenant-level access. The roadmap flags are different: they are instance-level rollout controls for incomplete community fork features.

## Decision

Add a separate experimental feature flag registry in `internal/featureflags`.

The Hub reads enabled experimental flags from `DISTR_EXPERIMENTAL_FEATURE_FLAGS` during `env.Initialize()`. The value is a comma, semicolon, whitespace, or newline separated list of known keys. The special value `all` enables every registered experimental flag. Unknown keys fail startup through the existing environment parsing path.

Expose the registry through `GET /api/v1/experimental-feature-flags`. The route requires authenticated organization context and admin role. The response contains all registered flags, their roadmap milestone, descriptions, and enabled state. The Angular `FeatureFlagService` reads the same endpoint and Organization Settings displays it read-only for administrators.

No database migration is added in PR-001. Persisted rollout state can be introduced by a later roadmap PR when per-organization, per-environment, or staged rollout behavior exists.

## Consequences

Future roadmap PRs can add API and UI guards without mixing experimental rollout controls into subscription features.

Enabled flags are instance-wide. This is intentionally simple for PR-001 and avoids creating a partial database-backed rollout model before the environment and lifecycle domains exist.

Operators must restart Hub after changing `DISTR_EXPERIMENTAL_FEATURE_FLAGS`.

## Alternatives Considered

Persist flags on `Organization.features`. This was rejected because those values currently model product entitlement and subscription behavior, not incomplete roadmap rollout controls.

Add a database table immediately. This was rejected because PR-001 only needs a framework and admin display; persisted rollout policy belongs with later governance and environment work.

## Validation

- `go test ./internal/featureflags`
- `go test ./internal/handlers -run TestExperimentalFeatureFlagResponses`
- `ng test`
- `mise run test`
- `mise run build:hub:community`
