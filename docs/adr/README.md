# Architecture Decision Records

Use this directory for architecture decision records when a change introduces or changes a public API, persistent data model, agent protocol, security boundary, or long-lived abstraction.

## Naming

Use a monotonic number and short kebab-case title:

```text
0001-release-bundle-immutability.md
0002-environment-lifecycle-model.md
```

## Template

```markdown
# ADR-0000: Title

## Status

Proposed | Accepted | Superseded

## Context

Describe the problem, constraints, existing behavior, and compatibility requirements.

## Decision

Describe the decision in concrete terms. Include API, data model, agent protocol, UI, and security implications when relevant.

## Consequences

Describe benefits, trade-offs, operational impact, migration needs, and follow-up work.

## Alternatives Considered

Describe other viable options and why they were not selected.

## Validation

List tests, migration checks, manual verification, and rollout or rollback notes.
```
