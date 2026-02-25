# Customer Portal UX Redesign — Trust & Compliance Dashboard

## Before you write any code

1. Read `CLAUDE.md` fully.
2. Explore `frontend/ui/src/app` and map the entire customer portal:
   - Every existing route and component (look for directories named `customer`, `portal`, `hub`, or similar)
   - The existing navigation shell (sidebar? top nav? layout component?)
   - All existing API service calls in the portal — what data is already being fetched?
   - The existing pages and what they show today
   - UI component library in use, icon library, styling approach (Tailwind? SCSS? CSS vars?)
   - Any existing types/DTOs for: deployments, application versions, artifacts, resources/attachments, agent status
3. Run `cat package.json` (frontend) to inventory all dependencies.
4. Identify specifically: how does the portal currently show the current deployed version? Deployment history? Agent status? Additional resources attached to versions?

This exploration is mandatory. Do not skip it. The entire goal is to surface existing data better — no new backend features, no new data models, no new API endpoints.

---

## What this is — and what it is not

**This is a UI/UX redesign.** The goal is to take data that already exists in the system and present it in a dramatically more useful and polished way. Every piece of information shown must come from an API that already exists.

**This is not:**

- A new feature build
- New backend logic or new API endpoints
- CVE/vulnerability scanning (future feature — placeholder UI only, clearly marked)
- Infrastructure questionnaire builder (future feature — do not build)
- Document upload (future feature — do not build)

The only functional changes permitted outside of UI layout work:

1. Make the customer portal navbar/branding customizable by the vendor (e.g., logo, primary color, portal name) — if this is not already supported, add it as a small config option
2. Graceful loading and empty states for all data (if not already done well)

**Future features** (mark as placeholder in the UI, do not implement):

- CVE/vulnerability scanning per artifact
- Infrastructure configuration questionnaire
- Compliance document upload by vendor

---

## The core UX problem to solve

Today the portal shows data. The upgrade is to make it tell a story:

> "You are running v2.4.1. The latest is v2.5.0. Your agent checked in 2 minutes ago and is healthy. Here are the resources your vendor attached to this version. Here's what changed since your last deployment."

This is what neither Docker Hub nor Artifact Hub can do — they are stateless discovery tools. They don't know what's running in your infrastructure. Distr does. That context should drive every design decision.

---

## Navigation structure

Examine what nav already exists and extend it. Conditional visibility rules:

| Section                   | Visible when                                       |
| ------------------------- | -------------------------------------------------- |
| **Home** (Compliance Hub) | Always                                             |
| **Deployments**           | Customer uses deployment agents                    |
| **Artifacts**             | Customer uses Helm/OCI artifact downloads          |
| **Configuration**         | BYOC / hands-off customer                          |
| **Security**              | Always (placeholder if no scan data exists yet)    |
| **Resources & Docs**      | Always (surfaces additional resources on versions) |
| **Setup & Installation**  | Always                                             |

Determine from the existing code whether this conditional logic is already in place or needs to be added.

---

## Page redesigns

### Home — Compliance Hub (primary deliverable, build fully)

**Layout inspiration:** Think Vercel dashboard meets Linear's activity feed. Not Docker Hub's flat list aesthetic, and not Artifact Hub's README-first layout. Dense information, clear hierarchy, purposeful color.

**The "deployment-aware" status bar at top (the key differentiator):**

```
[Your Application] ·  Running: v2.4.1  ·  Latest: v2.5.0 ↑ Update available  ·  Agent: ● Live (2 min ago)  ·  Last deployed: 3 days ago
```

This bar should be the first thing a customer sees. It immediately tells them the three things they care most about. Style it as a solid top strip, not a card.

**3-column card grid below:**

_Card 1 — Version & Deployment Health_

Show this data from existing deployment/version APIs:

- Currently running version (large, prominent)
- Version release date
- "X versions behind latest" indicator if applicable
- Mini deployment history: last 5 deployments as a compact timeline (version label + relative date)
- Agent last heartbeat with status dot (green/amber/red)
- Link: "View full deployment history →"

Do not invent data here. If a field doesn't come from an existing API, omit it or leave a `// TODO` comment.

_Card 2 — Available Resources_

This is the existing "additional resources" feature on application versions, surfaced better. Show it as:

- A clean list of files/links the vendor has attached to the current deployed version
- Each item: file type icon (PDF, ZIP, link), name, type label (e.g., "Release Notes", "Helm Chart"), and a download/open button
- Group by type if multiple exist
- "No resources attached to this version" empty state
- Link: "See all version resources →"

This replaces the concept of a "compliance document library" — it's just a better rendering of what already exists.

_Card 3 — Security (placeholder)_

Since CVE scanning is a future feature, this card should be:

- A clean, honest placeholder — not a fake data mock
- Show: "Vulnerability scanning coming soon"
- If SBOM files are attached as additional resources, surface them here: "SBOM available — Download SPDX / CycloneDX"
- A placeholder severity breakdown chart (greyed out, labelled "Available after scan integration")
- This card should look like a real card with a clear, honest "not yet available" state — not a disabled mess

**Activity feed (below cards):**

Pull from whatever event/audit data already exists in the API. Show a chronological feed of:

- Deployment events (version deployed, agent check-ins)
- New resources added to versions by vendor
- New application versions released

Each entry: event type icon + description + timestamp. Relative timestamps ("3 days ago") with tooltip showing exact datetime.
Take design inspiration from Linear's activity stream and Vercel's deployment log — monospaced timestamps, subtle separator lines, clean icon set.

---

### Deployments page (redesign existing page)

This page already exists — redesign it, don't rebuild from scratch.

**What to improve:**

- Make the "currently deployed" version visually prominent at the top, separate from the history table
- Version history table: add a visual "current" badge on the active row
- Add a version diff link between any two rows if release notes are available on the version
- Show agent name and status inline in the table
- Expandable row for deployment logs if that data exists
- Empty states and loading skeletons if not already polished

**Take inspiration from Docker Hub's tags table:** clean columns, digest/version as the anchor, status badge, timestamp. But add the "deployed to your environment" context they can't.

---

### Artifacts page (redesign existing page)

This page already exists — redesign it.

**What to improve:**

- Version selector at top: a dropdown that filters everything below (take inspiration from Artifact Hub's version selector pattern)
- For the selected version: show Helm chart download, image digest, pull command (copyable code block)
- Changelog/release notes section below if attached as a resource to the version
- A versions table below: version | release date | resources count | download — take inspiration from Docker Hub's tags table layout (clean, scannable, not too many columns)
- "Security report" column in the table should exist but show a placeholder badge ("Scan pending") since CVE scanning is a future feature

**The key UX improvement over Artifact Hub:** Artifact Hub's install modal is buried and awkward. Make the download/copy actions first-class: prominent copy buttons, clear `helm install` commands, no modal required for the primary action.

---

### Resources & Docs page (new section, light build)

This replaces a separate "Documents" and "Installation" section with one unified resource center.

**Two tabs:**

_Resources tab:_

- Card grid of all additional resources across all application versions, grouped by version
- Each card: file icon, name, version it belongs to, type label, upload date if available, download button
- Categories derived from file type or metadata, not hardcoded
- Empty state: "Your vendor hasn't attached any resources yet."

_Setup & Installation tab:_

- Installation and connection instructions (move here from wherever they currently live)
- API token display if applicable
- Links to external documentation

---

### Security page (future placeholder — build the skeleton only)

Build a clean, professional skeleton that communicates what this page will become.

- A prominent banner: "Vulnerability scanning will be available here once enabled."
- SBOM section: if SBOM files exist as version resources, surface download buttons here (SPDX, CycloneDX)
- A greyed-out CVE table structure showing the columns that will exist: CVE ID | Severity | CVSS | Component | Status | Affects current deployment — with a "No scan data available" overlay
- Do not populate this with mock CVE data. Keep it honest.

---

### Configuration page (BYOC customers only — light build)

- Agent connection status: large status indicator (connected/disconnected), last heartbeat, agent name/ID
- Current configuration snapshot: read-only code block display of whatever config is available via API
- Clean empty state if no config data exists

---

## Visual design direction

**Match the existing design system exactly.** Read the existing components carefully before writing a single line of CSS. Do not introduce new color palettes, new component patterns, or new utility classes.

Within the existing system, apply these principles to the redesigned pages:

- **Status colors must be semantic and consistent:** green = healthy/current, amber = warning/behind, red = error/critical. Do not use these colors decoratively.
- **Version numbers are first-class content.** They should look like version badges — monospace font, subtle background — not generic text.
- **Agent status dots** should pulse/animate if live, be static if stale.
- **The activity feed** should use a consistent left-border timeline pattern, not cards.
- **Download buttons** should be immediately actionable — no confirmation modals for file downloads.
- **Empty states** should be meaningful: explain why something is empty and what will appear when it's populated.

---

## Mock data

Populate every page with realistic mock data so stakeholders can react to the designs. Do not use placeholder text like "Lorem ipsum" or "Loading..." except in genuine loading skeletons.

Create `frontend/ui/src/app/customer-portal/mock-data.ts` (adjust path to match existing structure) with all mock datasets. Mark every usage with:

```ts
// MOCK DATA — replace with this.deploymentService.getXxx() when API is ready
```

**Mock datasets to include:**

Customer context:

- Name: "Acme Corp", environment: "Production", vendor: "DataStack Inc"
- `portalMode: 'byoc'` and `accessType: 'agent'` as top-level toggleable consts so you can preview both modes

Deployment state:

- Running version: v2.4.1 (released 2025-11-03)
- Latest available: v2.5.0 (released 2026-01-15) — so "update available" banner shows
- Deployment history: 6 entries across 8 months, including one rollback
- Agent: `prod-agent-us-east-1`, connected, last heartbeat 2 minutes ago

Version resources (additional resources on versions — this is existing functionality):

- On v2.4.1: "Release Notes v2.4.1" (markdown/link), "Helm Chart" (download), "SBOM (SPDX)" (file), "SBOM (CycloneDX)" (file)
- On v2.5.0: "Release Notes v2.5.0", "Helm Chart", "Upgrade Guide", "SOC2 Summary"
- On v2.3.0: "Release Notes v2.3.0", "Hardening Guide v2.0"

Activity feed (10–12 realistic entries):

- 2 days ago: Deployment — "v2.4.1 deployed successfully via prod-agent-us-east-1"
- 8 days ago: Resource — "DataStack Inc added Upgrade Guide to v2.5.0"
- 15 days ago: Deployment — "v2.4.0 deployed successfully"
- 18 days ago: Release — "v2.5.0 released by DataStack Inc"
- 28 days ago: Deployment — "Rollback to v2.3.9 initiated"
- 35 days ago: Resource — "DataStack Inc added SBOM (SPDX) to v2.4.1"
- 42 days ago: Deployment — "v2.4.0 deployed successfully"
- 50 days ago: Release — "v2.4.1 released by DataStack Inc"
- 60 days ago: Deployment — "v2.3.9 deployed successfully"
- 75 days ago: Release — "v2.4.0 released by DataStack Inc"

Artifacts (for artifact-mode view):

- 5 versions with Helm chart pull commands, image digests (sha256:...), changelog snippets

---

## Implementation constraints

- Follow existing component, service, and routing patterns exactly — read them before writing
- No new npm dependencies unless already installed
- All new routes added to existing router config, not a new file
- Use existing auth guard pattern
- If an existing portal shell/layout exists, add pages as children — do not create a new shell
- TypeScript: match the strictness of the existing codebase
- No new API endpoints — all data comes from APIs that currently exist

---

## Deliverable

1. All modified and new Angular components, following existing conventions
2. Brief comment at top of each new file: what it does, what API data it expects
3. `// TODO: replace with real API call` everywhere mock data is used
4. `PORTAL_CHANGES.md` in the portal directory summarizing: what was redesigned, what was added, what future API integrations are needed, and where the three future features (CVE scanning, questionnaire builder, document upload) will slot in once built
