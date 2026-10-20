# Agent Instructions

- Import Astro components and TypeScript data via `~/`, not relative paths.
- Never add files to `.prettierignore` or similar as a shortcut.
- Use `pnpm run`, never `npm` or `yarn`.
- `pnpm format` deletes unreferenced images in `src/assets`. Reference an image before adding it; `pnpm images:unused` lists orphans.
- Keep example files (Compose, `.env`, Helm, Dockerfile, configs) out of fenced code blocks. Save each one under its real name in a folder named after the page, import it with `?raw` and render it with `<Code>`.
- Give blog subheadings a specific topic, e.g. "How Traefik and Docker Swarm routing works" instead of "How It Works".
