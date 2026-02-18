# Database Test Data Requirements

This document outlines the database records that need to be seeded to make deployments and artifacts appear in both the vendor and customer portals.

## Overview

The mock data in `frontend/ui/src/app/components/home/mock-data.ts` represents the **shape** of data that should exist in the database. This data is currently only used on the customer portal home page for UI display purposes.

For the actual `/deployments` and `/artifacts` pages to show data, you need to seed the PostgreSQL database with proper records.

## Required Database Tables and Sample Data

### 1. Deployments (`deployments` table)

Based on `MOCK_DEPLOYMENTS` from mock-data.ts, you need:

```sql
-- Sample deployments for the customer portal
INSERT INTO deployments (
  id,
  deployment_target_id,
  application_version_id,
  created_at,
  updated_at
) VALUES
  -- Deployment 1: DataStack Platform v2.4.1
  (gen_random_uuid(), '<deployment_target_id>', '<app_version_id_2.4.1>', NOW() - INTERVAL '3 days', NOW()),

  -- Deployment 2: Analytics Engine v1.8.3
  (gen_random_uuid(), '<deployment_target_id>', '<app_version_id_1.8.3>', NOW() - INTERVAL '15 minutes', NOW()),

  -- Deployment 3: Monitoring Dashboard v3.2.1
  (gen_random_uuid(), '<deployment_target_id>', '<app_version_id_3.2.1>', NOW() - INTERVAL '7 days', NOW());
```

### 2. Deployment Targets (`deployment_targets` table)

You need at least one deployment target (agent):

```sql
INSERT INTO deployment_targets (
  id,
  name,
  customer_organization_id,
  created_at,
  updated_at
) VALUES
  (gen_random_uuid(), 'prod-agent-us-east-1', '<customer_org_id>', NOW() - INTERVAL '90 days', NOW());
```

### 3. Deployment Status (`deployment_log_records` table)

For each deployment, add status records:

```sql
-- For DataStack Platform (healthy, running)
INSERT INTO deployment_log_records (
  id,
  deployment_id,
  type,
  message,
  created_at
) VALUES
  (gen_random_uuid(), '<deployment_1_id>', 'healthy', 'All components running normally', NOW() - INTERVAL '2 minutes');

-- For Analytics Engine (healthy, updating)
INSERT INTO deployment_log_records (
  id,
  deployment_id,
  type,
  message,
  created_at
) VALUES
  (gen_random_uuid(), '<deployment_2_id>', 'progressing', 'Updating from v1.8.3 to v1.9.0', NOW() - INTERVAL '1 minute');

-- For Monitoring Dashboard (degraded)
INSERT INTO deployment_log_records (
  id,
  deployment_id,
  type,
  message,
  created_at
) VALUES
  (gen_random_uuid(), '<deployment_3_id>', 'running', 'Some pods restarting frequently', NOW() - INTERVAL '5 minutes');
```

### 4. Applications (`applications` table)

Create applications that will have deployments:

```sql
INSERT INTO applications (
  id,
  organization_id,
  name,
  created_at,
  updated_at
) VALUES
  (gen_random_uuid(), '<vendor_org_id>', 'DataStack Platform', NOW() - INTERVAL '1 year', NOW()),
  (gen_random_uuid(), '<vendor_org_id>', 'Analytics Engine', NOW() - INTERVAL '1 year', NOW()),
  (gen_random_uuid(), '<vendor_org_id>', 'Monitoring Dashboard', NOW() - INTERVAL '1 year', NOW());
```

### 5. Application Versions (`application_versions` table)

Create versions for each application:

```sql
-- DataStack Platform versions
INSERT INTO application_versions (
  id,
  application_id,
  name,
  created_at
) VALUES
  (gen_random_uuid(), '<datastack_app_id>', 'v2.4.1', NOW() - INTERVAL '35 days'),
  (gen_random_uuid(), '<datastack_app_id>', 'v2.5.0', NOW() - INTERVAL '5 days');

-- Analytics Engine versions
INSERT INTO application_versions (
  id,
  application_id,
  name,
  created_at
) VALUES
  (gen_random_uuid(), '<analytics_app_id>', 'v1.8.3', NOW() - INTERVAL '60 days'),
  (gen_random_uuid(), '<analytics_app_id>', 'v1.9.0', NOW() - INTERVAL '10 days');

-- Monitoring Dashboard versions
INSERT INTO application_versions (
  id,
  application_id,
  name,
  created_at
) VALUES
  (gen_random_uuid(), '<monitoring_app_id>', 'v3.2.1', NOW() - INTERVAL '42 days');
```

### 6. Artifacts (`artifacts` table)

Based on `MOCK_ARTIFACTS` from mock-data.ts:

```sql
INSERT INTO artifacts (
  id,
  organization_id,
  name,
  image_url,
  created_at,
  updated_at
) VALUES
  (gen_random_uuid(), '<vendor_org_id>', 'datastack/platform', NULL, NOW() - INTERVAL '2 years', NOW()),
  (gen_random_uuid(), '<vendor_org_id>', 'datastack/analytics-engine', NULL, NOW() - INTERVAL '2 years', NOW()),
  (gen_random_uuid(), '<vendor_org_id>', 'datastack/monitoring', NULL, NOW() - INTERVAL '2 years', NOW()),
  (gen_random_uuid(), '<vendor_org_id>', 'datastack/cli-tools', NULL, NOW() - INTERVAL '2 years', NOW());
```

### 7. Artifact Versions/Tags (OCI registry data)

For each artifact, you need to push actual OCI images/charts to the registry, which will create records in the registry tables. The frontend queries these through the `ArtifactsService`.

Example using Docker:

```bash
# Tag and push container images
docker tag my-image:v2.5.0 registry.example.com/datastack/platform:v2.5.0
docker push registry.example.com/datastack/platform:v2.5.0

docker tag analytics:v1.9.0 registry.example.com/datastack/analytics-engine:v1.9.0
docker push registry.example.com/datastack/analytics-engine:v1.9.0

# Push Helm chart
helm package ./monitoring-chart
helm push monitoring-3.2.1.tgz oci://registry.example.com/datastack/monitoring
```

### 8. Organizations

You need both vendor and customer organizations:

```sql
-- Vendor organization
INSERT INTO organizations (
  id,
  slug,
  name,
  type,
  created_at
) VALUES
  (gen_random_uuid(), 'datastack-vendor', 'DataStack Inc', 'vendor', NOW() - INTERVAL '2 years');

-- Customer organization
INSERT INTO customer_organizations (
  id,
  organization_id,
  name,
  created_at
) VALUES
  (gen_random_uuid(), '<vendor_org_id>', 'Acme Corp', NOW() - INTERVAL '1 year');
```

## Data Relationships

```
Organizations
  ├── Applications (vendor creates)
  │   └── ApplicationVersions
  │       └── Deployments (customer deploys)
  │           └── DeploymentLogRecords (status updates)
  └── Artifacts (vendor pushes)
      └── ArtifactTags/Versions (OCI registry)

DeploymentTargets (customer agents)
  └── Deployments
  └── CurrentStatus (heartbeat)
```

## How The UI Consumes This Data

### Customer Portal Home Page (`/home`)

- Uses `DeploymentTargetsService.list()` to get deployment targets with their deployments
- The "Version & Deployment" card shows:
  - First deployment's current version
  - Agent connection status
  - List of all deployments with health indicators (from `mockDeployments()` - **currently using mock data**)

### Vendor Portal Dashboard (`/dashboard`)

- Uses `DeploymentTargetsService.list()` to show all customer agents
- Uses `ApplicationsService.list()` to show applications
- Uses `ArtifactsService.list()` to show artifacts

### Deployments Page (`/deployments`)

- Uses `DeploymentTargetsService.list()` to get all deployment targets
- Uses `DeploymentTargetsMetricsService` for metrics
- Shows deployment cards with status, metrics, and actions

### Artifacts Page (`/artifacts`)

- Uses `ArtifactsService.list()` to get all artifacts
- Shows artifact cards with versions, downloads, and metadata

## Mock Data vs Real Data

**Important**: The `MOCK_DEPLOYMENTS` and `MOCK_ARTIFACTS` in `mock-data.ts` are **only used on the customer portal home page** for the "Version & Deployment" card. They do NOT populate the actual `/deployments` or `/artifacts` pages.

To see data on those pages, you must:

1. Seed the database with the SQL above
2. Push actual OCI artifacts to the registry
3. The services will fetch this data via API calls

## Recommended Seeding Approach

Create a database migration or seed script:

- `internal/migrations/sql/999_seed_test_data.up.sql`
- Or use `cmd/hub/cmd/seed.go` if it exists
- Or create a test data generator in Go that uses the existing DB package

The seed data should create a complete scenario:

- 1 vendor organization (DataStack Inc)
- 1 customer organization (Acme Corp)
- 3 applications with multiple versions each
- 4 artifacts
- 1 deployment target (agent)
- 3 active deployments with various health states
