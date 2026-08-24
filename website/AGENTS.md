# Agent Instructions

- For imports of Astro components or TypeScript data, always use `~/` and not relative paths.
- Don't do any shortcuts by adding files to `.prettierignore` or similar.
- Always use `pnpm` for package management and execute commands using `pnpm run`, don't use `npm` or `yarn`.
- `pnpm format` deletes every image in `src/assets` that no page, post or component references (`pnpm images:prune`). Reference an image before you add it, and use `pnpm images:unused` to list orphans without deleting them.
- Blog subheadings should always include a specific reference to the topic rather than using generic titles like "Architecture", "How It Works", or "Implementation". For example, use "How Traefik and Docker Swarm routing works" instead of "How It Works", or "Basic Docker Swarm routing architecture" instead of "Architecture".
