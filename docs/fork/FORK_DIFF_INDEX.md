# Fork Diff Index

This file tracks generic fork additions and upstream-facing changes introduced after the upstream base recorded in `docs/fork/UPSTREAM_BASE.md`.

## Current Status

PR-000 through PR-002 are implemented locally. PR-002 adds the feature-flagged Environment domain model without changing existing deployment target, deployment, or agent behavior.

## Tracking Template

Use one entry per pull request:

```markdown
## PR-000 - Baseline and fork records

- Status:
- Upstream base:
- Feature flag:
- User-facing behavior:
- Database changes:
- API changes:
- UI changes:
- Agent protocol changes:
- Documentation:
- Tests:
- Upstream contribution notes:
- Compatibility notes:
```

## Entries

### PR-000 - Baseline and fork records

- Status: Complete for documentation-only baseline; local build limitations are recorded in `docs/fork/UPSTREAM_BASE.md`.
- Upstream base: `b49fb27eb6270d7a71eed82b12e47eec1217c4cf`
- Feature flag: None
- User-facing behavior: None
- Database changes: None
- API changes: None
- UI changes: None
- Agent protocol changes: None
- Documentation: Added fork baseline, fork diff index, ADR template, root Codex instructions, and master roadmap.
- Tests: See `docs/fork/UPSTREAM_BASE.md`.
- Upstream contribution notes: Documentation-only fork baseline.
- Compatibility notes: No runtime compatibility impact.

### PR-001 - Experimental feature flag framework

- Status: Implemented locally; focused tests, builds, and diff-scoped lint passed.
- Upstream base: `b49fb27eb6270d7a71eed82b12e47eec1217c4cf`
- Feature flag: Added instance-level experimental flags configured by `DISTR_EXPERIMENTAL_FEATURE_FLAGS`.
- User-facing behavior: Admins can view registered experimental roadmap flags and enabled state in Organization Settings.
- Database changes: None.
- API changes: Added `GET /api/v1/experimental-feature-flags`, admin-only.
- UI changes: Added readonly Experimental features table on Organization Settings and Angular service support.
- Agent protocol changes: None.
- Documentation: Added PR-001 user story/API notes and ADR-0001.
- Tests: `mise run test`, Angular `ng test`, `mise run build:hub:community`, Docker agent build, Kubernetes agent build, direct migration validation, touched-file Prettier check, and diff-scoped Go lint passed.
- Upstream contribution notes: Community-neutral abstraction; no adopter-specific logic.
- Compatibility notes: Existing deployments and agents are unchanged. Unknown configured flag keys are rejected at Hub startup.

### PR-002 - Environment domain model

- Status: Implemented locally; backend, frontend, migration, lint, and build verification completed.
- Upstream base: `b49fb27eb6270d7a71eed82b12e47eec1217c4cf`
- Feature flag: Uses `DISTR_EXPERIMENTAL_FEATURE_FLAGS=environments`.
- User-facing behavior: Admins can manage Environments from a new feature-flagged Environments page.
- Database changes: Added organization-scoped `Environment` table with unique names per organization and non-negative sort order.
- API changes: Added feature-flagged CRUD endpoints under `/api/v1/environments`.
- UI changes: Added Environments route, sidebar link, Angular service, and CRUD table/dialog UI with loading, error, empty, create, update, and delete states.
- Agent protocol changes: None.
- Documentation: Added PR-002 notes and ADR-0002.
- Tests: `go test -p=1 ./...`, Angular `ng test --watch=false`, `pnpm run build:community`, hub community binary build, Docker agent build, Kubernetes agent build, direct migration validation, touched-file Prettier check, and diff-scoped Go lint passed. Full repo `pnpm lint` and full repo Go lint still report pre-existing formatting issues outside the PR-owned diff.
- Upstream contribution notes: Community-neutral environment model; no adopter-specific logic.
- Compatibility notes: Existing deployment targets, deployments, and agents are unchanged. No target-to-environment assignment is added in PR-002.
