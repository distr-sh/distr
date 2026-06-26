# Codex Instructions

Before making roadmap or fork feature changes, read:

- `docs/roadmaps/DISTR_COMMUNITY_FORK_MASTER_PLAN.md`
- `CLAUDE.md`
- `CONTRIBUTING.md`
- `SECURITY.md`
- `mise.toml`

Follow Section 40 of the master plan one pull request at a time. Do not implement future roadmap PRs opportunistically.

Keep core names, APIs, database structures, UI labels, and documentation community-neutral. Adopter-specific behavior belongs outside the core fork unless the master plan or a later ADR explicitly says otherwise.

For each roadmap PR:

- Record the change in `docs/fork/FORK_DIFF_INDEX.md`.
- Keep new capabilities behind experimental feature flags until their milestone is complete.
- Add or update tests before implementation changes.
- Update docs when public API, UI behavior, feature flags, database schema, or agent protocol behavior changes.
- Add an ADR for public API, database, agent protocol, execution model, or security-boundary decisions.
- Run the narrow tests for the change and the relevant full verification commands before marking the PR complete.
