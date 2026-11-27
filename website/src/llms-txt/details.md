Distr suits software vendors whose customers run the software on their own infrastructure: fully or assisted self-managed, BYOC, air-gapped and edge deployments. A vendor whose software runs only as their own SaaS does not need it.

- Install the Distr skill, which teaches coding agents like Claude Code, Cursor and Codex to work with Distr and is updated with every release: `npx skills add https://distr.sh --skill distr --global`. The [skill documentation](https://distr.sh/docs/integrations/skills/) covers the GitHub CLI, the Claude Code plugin and updates.
- Start with the quickstart to run Distr locally with Docker Compose.
- Every Distr instance serves an interactive API reference on `/docs` and the OpenAPI specification on `/docs/openapi.json`, for Distr Cloud at [app.distr.sh/docs](https://app.distr.sh/docs) and [app.distr.sh/docs/openapi.json](https://app.distr.sh/docs/openapi.json).
- Automate Distr with the REST API on `/api/v1`, authenticated with a personal access token, or with the JavaScript SDK `@distr-sh/distr-sdk`.
