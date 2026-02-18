# Customer Portal Redesign — Changes Summary

## What Was Redesigned

### Home Page (`/home`) — Compliance Hub
Previously a minimal page showing only branding markdown. Now a full compliance dashboard:

- **Deployment-aware status bar** at top: shows running version, latest version, update availability, agent status with live/stale/disconnected indicator, and last deployment time. Uses real data from `DeploymentTargetsService`.
- **3-column card grid**:
  - **Version & Deployment Health**: current version badge with status, version-behind indicator, agent heartbeat, mini deployment history timeline
  - **Available Resources**: lists files/links attached to the current deployed version (release notes, Helm charts, SBOMs, guides) with download/open actions
  - **Security (placeholder)**: honest "coming soon" state for CVE scanning, greyed-out severity breakdown, SBOM download buttons if available
- **Activity feed**: chronological timeline of deployment events, resource additions, and version releases with relative timestamps

### Sidebar Navigation (customer section)
Extended with new entries, each with conditional visibility:

| Nav Item | Condition |
|---|---|
| Home | Always visible |
| Deployments | `deployment_targets` feature enabled |
| Artifacts | `artifacts` feature enabled |
| Configuration | `deployment_targets` feature enabled |
| Security | Always visible |
| Resources & Docs | Always visible |
| Users | Admin role only |
| Secrets | `deployment_targets` feature enabled |

## What Was Added (New Pages)

### Security Page (`/security`)
- `security-page.component.ts`
- Placeholder page for future vulnerability scanning
- Prominent banner explaining the feature will be available once enabled
- SBOM section: surfaces SBOM files from version resources for download
- Greyed-out CVE table structure showing future column layout with "No scan data available" overlay
- No mock CVE data — honest placeholder

### Resources & Docs Page (`/resources`)
- `resources-page.component.ts`
- Two tabs: Resources and Setup & Installation
- Resources tab: card grid of all additional resources across all application versions, grouped by version
- Setup & Installation tab: agent installation instructions, access token management link
- Empty state for when no resources exist

### Configuration Page (`/configuration`)
- `configuration-page.component.ts`
- Agent connection status with large visual indicator (green/amber/gray)
- Agent details: name, type, version, last heartbeat
- Active deployments list with status indicators
- Uses real data from `DeploymentTargetsService`

### Mock Data File
- `mock-data.ts`
- Contains all mock datasets for: customer context, deployment state, deployment history, version resources, activity feed, artifact versions
- Every usage is marked with `// MOCK DATA — replace with ...` comments

## What Was Modified

| File | Change |
|---|---|
| `home.component.ts` | Complete rewrite from branding-only to Compliance Hub |
| `home.component.html` | Complete rewrite with status bar, card grid, activity feed |
| `side-bar.component.ts` | Added `faShieldHalved`, `faFileLines` icons |
| `side-bar.component.html` | Added Security, Resources & Docs, Configuration nav items |
| `app-logged-in.routes.ts` | Added `/security`, `/resources`, `/configuration` routes |

## Future API Integrations Needed

All `// MOCK DATA` and `// TODO` markers indicate where real API integration is needed:

1. **Activity feed**: No existing audit/event log API. Will need a new endpoint like `GET /api/v1/activity` to replace `MOCK_ACTIVITY_FEED`.
2. **Version comparison** ("X versions behind latest"): Requires comparing deployed version against available versions from `ApplicationsService`. Currently uses mock data for the "latest version" display.
3. **Version release dates**: `ApplicationVersion.createdAt` exists but isn't prominently surfaced. No mock replacement needed — just wire it.
4. **Resources per version**: `ApplicationsService.getResources()` already exists. Mock data should be replaced with real calls passing the current deployed version's application and version IDs.

## Future Features — Where They Slot In

### CVE/Vulnerability Scanning
- **Security page** (`security-page.component.ts`): the greyed-out CVE table is ready to be populated. Replace the overlay and skeleton rows with real scan data.
- **Home page** Security card: update severity breakdown from placeholder to real counts.
- **Artifacts page**: add a "Security report" column to the versions table (mentioned in spec, not yet built since this is artifact page redesign scope).

### Infrastructure Configuration Questionnaire
- Would be a new page, likely at `/questionnaire` or as a tab within the Configuration page.
- Add a new sidebar nav item conditionally visible based on a new feature flag.

### Compliance Document Upload by Vendor
- Would extend the Resources page with an upload section.
- Backend would need a new endpoint for vendor-initiated file uploads attached to versions.
- The existing `ApplicationVersionResource` model may need extension for file uploads vs markdown content.
