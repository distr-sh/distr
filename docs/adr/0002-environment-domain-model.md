# ADR-0002 - Environment Domain Model

## Status

Accepted for PR-002.

## Context

The roadmap introduces Environments as a promotion-stage or operational-purpose grouping used later by lifecycles, release bundles, and deployment planning. PR-002 needs the standalone domain model without changing existing deployment target execution.

## Decision

Add an organization-scoped `Environment` table, DB type, API DTOs, CRUD API, and admin UI. The API is guarded by the `environments` experimental feature flag introduced in PR-001.

The first model includes the roadmap fields that do not require later domains:

- name;
- description;
- sort order;
- production marker;
- dynamic target allowance;
- nullable retention policy ID placeholder.

Deployment targets are not assigned to environments in PR-002. That keeps existing deployment target behavior unchanged and leaves target assignment rules for later planner/lifecycle work.

## Consequences

Existing deployments and agents are unaffected. Environments can be created and managed independently, but they do not yet influence promotion eligibility, deployment planning, target selection, or retention.

The `retention_policy_id` column is intentionally nullable and has no foreign key until retention policies are introduced.
