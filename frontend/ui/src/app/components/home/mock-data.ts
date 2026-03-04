// Mock data for the customer portal compliance hub.
// All mock data here simulates what would come from existing API endpoints.
// Each usage site is marked with: // MOCK DATA — replace with real API call

export type PortalMode = 'byoc' | 'managed';
export type AccessType = 'agent' | 'artifact';

export const PORTAL_MODE: PortalMode = 'byoc';
export const ACCESS_TYPE: AccessType = 'agent';

export interface MockDeploymentEntry {
  version: string;
  date: string;
  status: 'success' | 'rollback' | 'failed';
  agent: string;
}

export interface MockResource {
  name: string;
  type: 'release-notes' | 'helm-chart' | 'sbom-spdx' | 'sbom-cyclonedx' | 'guide' | 'document' | 'cve-report';
  format: 'markdown' | 'download' | 'link';
  version: string;
  url?: string;
  cveData?: MockCVEReport;
}

export interface MockCVE {
  id: string;
  severity: 'critical' | 'high' | 'medium' | 'low';
  component: string;
  version: string;
  fixedVersion?: string;
  description: string;
  publishedDate: string;
}

export interface MockCVEReport {
  reportDate: string;
  totalCVEs: number;
  critical: number;
  high: number;
  medium: number;
  low: number;
  cves: MockCVE[];
}

export interface MockActivityEntry {
  type: 'deployment' | 'resource' | 'release';
  description: string;
  timestamp: string;
}

export interface MockArtifactVersion {
  version: string;
  releaseDate: string;
  digest: string;
  helmPullCommand: string;
  changelog: string;
  resourceCount: number;
}

export const MOCK_CUSTOMER = {
  name: 'Acme Corp',
  environment: 'Production',
  vendor: 'DataStack Inc',
};

export const MOCK_DEPLOYMENT_STATE = {
  currentVersion: 'v2.4.1',
  currentVersionDate: '2025-11-03T00:00:00Z',
  latestVersion: 'v2.5.0',
  latestVersionDate: '2026-01-15T00:00:00Z',
  versionsBehind: 1,
  agentName: 'prod-agent-us-east-1',
  agentConnected: true,
  agentLastHeartbeat: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
  lastDeployedAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
};

export const MOCK_DEPLOYMENT_HISTORY: MockDeploymentEntry[] = [
  {
    version: 'v2.4.1',
    date: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
    status: 'success',
    agent: 'prod-agent-us-east-1',
  },
  {
    version: 'v2.4.0',
    date: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000).toISOString(),
    status: 'success',
    agent: 'prod-agent-us-east-1',
  },
  {
    version: 'v2.3.9',
    date: new Date(Date.now() - 28 * 24 * 60 * 60 * 1000).toISOString(),
    status: 'rollback',
    agent: 'prod-agent-us-east-1',
  },
  {
    version: 'v2.3.9',
    date: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
    status: 'success',
    agent: 'prod-agent-us-east-1',
  },
  {
    version: 'v2.3.8',
    date: new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString(),
    status: 'success',
    agent: 'prod-agent-us-east-1',
  },
  {
    version: 'v2.3.7',
    date: new Date(Date.now() - 120 * 24 * 60 * 60 * 1000).toISOString(),
    status: 'success',
    agent: 'prod-agent-us-east-1',
  },
];

export const MOCK_RESOURCES: MockResource[] = [
  {name: 'Release Notes v2.4.1', type: 'release-notes', format: 'markdown', version: 'v2.4.1'},
  {name: 'Helm Chart', type: 'helm-chart', format: 'download', version: 'v2.4.1'},
  {name: 'SBOM (SPDX)', type: 'sbom-spdx', format: 'download', version: 'v2.4.1'},
  {name: 'SBOM (CycloneDX)', type: 'sbom-cyclonedx', format: 'download', version: 'v2.4.1'},
  {
    name: 'CVE Report v2.4.1',
    type: 'cve-report',
    format: 'download',
    version: 'v2.4.1',
    cveData: {
      reportDate: '2025-11-10T00:00:00Z',
      totalCVEs: 8,
      critical: 1,
      high: 2,
      medium: 3,
      low: 2,
      cves: [
        {
          id: 'CVE-2025-1234',
          severity: 'critical',
          component: 'openssl',
          version: '3.0.8',
          fixedVersion: '3.0.12',
          description: 'Critical buffer overflow vulnerability in OpenSSL allowing remote code execution',
          publishedDate: '2025-10-15T00:00:00Z',
        },
        {
          id: 'CVE-2025-2345',
          severity: 'high',
          component: 'nodejs',
          version: '18.16.0',
          fixedVersion: '18.19.0',
          description: 'Prototype pollution vulnerability in Node.js runtime',
          publishedDate: '2025-09-22T00:00:00Z',
        },
        {
          id: 'CVE-2025-3456',
          severity: 'high',
          component: 'postgresql-client',
          version: '15.3',
          fixedVersion: '15.5',
          description: 'SQL injection vulnerability in PostgreSQL client library',
          publishedDate: '2025-08-05T00:00:00Z',
        },
        {
          id: 'CVE-2025-4567',
          severity: 'medium',
          component: 'express',
          version: '4.18.2',
          fixedVersion: '4.19.1',
          description: 'Path traversal vulnerability in Express middleware',
          publishedDate: '2025-07-12T00:00:00Z',
        },
        {
          id: 'CVE-2025-5678',
          severity: 'medium',
          component: 'axios',
          version: '1.4.0',
          fixedVersion: '1.6.2',
          description: 'SSRF vulnerability in axios HTTP client',
          publishedDate: '2025-06-20T00:00:00Z',
        },
        {
          id: 'CVE-2025-6789',
          severity: 'medium',
          component: 'lodash',
          version: '4.17.20',
          fixedVersion: '4.17.21',
          description: 'Command injection via template strings in lodash',
          publishedDate: '2025-05-08T00:00:00Z',
        },
        {
          id: 'CVE-2025-7890',
          severity: 'low',
          component: 'moment',
          version: '2.29.3',
          description: 'Regular expression DoS in moment.js date parsing (no fix available)',
          publishedDate: '2025-04-15T00:00:00Z',
        },
        {
          id: 'CVE-2025-8901',
          severity: 'low',
          component: 'winston',
          version: '3.8.2',
          fixedVersion: '3.11.0',
          description: 'Information disclosure through verbose error logging',
          publishedDate: '2025-03-10T00:00:00Z',
        },
      ],
    },
  },
  {name: 'Release Notes v2.5.0', type: 'release-notes', format: 'markdown', version: 'v2.5.0'},
  {name: 'Helm Chart', type: 'helm-chart', format: 'download', version: 'v2.5.0'},
  {name: 'Upgrade Guide', type: 'guide', format: 'markdown', version: 'v2.5.0'},
  {name: 'SOC2 Summary', type: 'document', format: 'download', version: 'v2.5.0'},
  {
    name: 'CVE Report v2.5.0',
    type: 'cve-report',
    format: 'download',
    version: 'v2.5.0',
    cveData: {
      reportDate: '2026-01-20T00:00:00Z',
      totalCVEs: 3,
      critical: 0,
      high: 1,
      medium: 1,
      low: 1,
      cves: [
        {
          id: 'CVE-2025-9012',
          severity: 'high',
          component: 'helm',
          version: '3.12.0',
          fixedVersion: '3.13.2',
          description: 'Privilege escalation in Helm chart installation',
          publishedDate: '2025-12-05T00:00:00Z',
        },
        {
          id: 'CVE-2026-0123',
          severity: 'medium',
          component: 'redis',
          version: '7.0.11',
          fixedVersion: '7.2.3',
          description: 'Memory leak in Redis connection pool',
          publishedDate: '2025-11-18T00:00:00Z',
        },
        {
          id: 'CVE-2026-0234',
          severity: 'low',
          component: 'yaml',
          version: '2.2.2',
          fixedVersion: '2.3.4',
          description: 'Denial of service via malformed YAML input',
          publishedDate: '2025-10-25T00:00:00Z',
        },
      ],
    },
  },
  {name: 'Release Notes v2.3.0', type: 'release-notes', format: 'markdown', version: 'v2.3.0'},
  {name: 'Hardening Guide v2.0', type: 'guide', format: 'download', version: 'v2.3.0'},
];

export const MOCK_ACTIVITY_FEED: MockActivityEntry[] = [
  {
    type: 'deployment',
    description: 'v2.4.1 deployed successfully via prod-agent-us-east-1',
    timestamp: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'resource',
    description: 'DataStack Inc added Upgrade Guide to v2.5.0',
    timestamp: new Date(Date.now() - 8 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'deployment',
    description: 'v2.4.0 deployed successfully',
    timestamp: new Date(Date.now() - 15 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'release',
    description: 'v2.5.0 released by DataStack Inc',
    timestamp: new Date(Date.now() - 18 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'deployment',
    description: 'Rollback to v2.3.9 initiated',
    timestamp: new Date(Date.now() - 28 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'resource',
    description: 'DataStack Inc added SBOM (SPDX) to v2.4.1',
    timestamp: new Date(Date.now() - 35 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'deployment',
    description: 'v2.4.0 deployed successfully',
    timestamp: new Date(Date.now() - 42 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'release',
    description: 'v2.4.1 released by DataStack Inc',
    timestamp: new Date(Date.now() - 50 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'deployment',
    description: 'v2.3.9 deployed successfully',
    timestamp: new Date(Date.now() - 60 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    type: 'release',
    description: 'v2.4.0 released by DataStack Inc',
    timestamp: new Date(Date.now() - 75 * 24 * 60 * 60 * 1000).toISOString(),
  },
];

export const MOCK_ARTIFACT_VERSIONS: MockArtifactVersion[] = [
  {
    version: 'v2.5.0',
    releaseDate: '2026-01-15T00:00:00Z',
    digest: 'sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4',
    helmPullCommand: 'helm pull oci://registry.datastack.io/charts/datastack --version 2.5.0',
    changelog: 'Added new monitoring dashboard, improved query performance by 40%, fixed connection pooling issue.',
    resourceCount: 4,
  },
  {
    version: 'v2.4.1',
    releaseDate: '2025-11-03T00:00:00Z',
    digest: 'sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',
    helmPullCommand: 'helm pull oci://registry.datastack.io/charts/datastack --version 2.4.1',
    changelog: 'Patch release: fixed critical auth token refresh bug, updated dependencies.',
    resourceCount: 4,
  },
  {
    version: 'v2.4.0',
    releaseDate: '2025-09-20T00:00:00Z',
    digest: 'sha256:7c92a1355c57214bc7eb4cb67ff0fc15aff0c0a0b9db8d9a62ab4db218e5c4f3',
    helmPullCommand: 'helm pull oci://registry.datastack.io/charts/datastack --version 2.4.0',
    changelog: 'Added RBAC improvements, new audit log viewer, PostgreSQL 16 support.',
    resourceCount: 2,
  },
  {
    version: 'v2.3.9',
    releaseDate: '2025-07-12T00:00:00Z',
    digest: 'sha256:5d20c808b198c8a1e9a9f2baf7e7e6c3db01b0a6e71ff3c8a4b38e232c5b7a1d',
    helmPullCommand: 'helm pull oci://registry.datastack.io/charts/datastack --version 2.3.9',
    changelog: 'Security patch for CVE-2025-1234, performance improvements for large datasets.',
    resourceCount: 1,
  },
  {
    version: 'v2.3.0',
    releaseDate: '2025-04-01T00:00:00Z',
    digest: 'sha256:2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824',
    helmPullCommand: 'helm pull oci://registry.datastack.io/charts/datastack --version 2.3.0',
    changelog: 'Major release: new deployment pipeline, Kubernetes 1.29 support, redesigned admin UI.',
    resourceCount: 2,
  },
];

// Extended deployment and artifact data structures for home page display
export interface MockDeployment {
  id: string;
  applicationName: string;
  currentVersion: string;
  targetVersion?: string;
  status: 'running' | 'updating' | 'failed' | 'pending';
  health: 'healthy' | 'degraded' | 'unhealthy';
  agentName: string;
  agentConnected: boolean;
  lastDeployedAt: string;
  lastHeartbeat: string;
  environment: string;
}

export interface MockArtifact {
  id: string;
  name: string;
  type: 'container-image' | 'helm-chart' | 'generic';
  latestVersion: string;
  totalVersions: number;
  totalDownloads: number;
  lastPushedAt: string;
  size: number;
}

export const MOCK_DEPLOYMENTS: MockDeployment[] = [
  {
    id: 'dep-1',
    applicationName: 'DataStack Platform',
    currentVersion: 'v2.4.1',
    status: 'running',
    health: 'healthy',
    agentName: 'prod-agent-us-east-1',
    agentConnected: true,
    lastDeployedAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
    lastHeartbeat: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
    environment: 'Production',
  },
  {
    id: 'dep-2',
    applicationName: 'Analytics Engine',
    currentVersion: 'v1.8.3',
    targetVersion: 'v1.9.0',
    status: 'updating',
    health: 'healthy',
    agentName: 'prod-agent-us-east-1',
    agentConnected: true,
    lastDeployedAt: new Date(Date.now() - 15 * 60 * 1000).toISOString(),
    lastHeartbeat: new Date(Date.now() - 1 * 60 * 1000).toISOString(),
    environment: 'Production',
  },
  {
    id: 'dep-3',
    applicationName: 'Monitoring Dashboard',
    currentVersion: 'v3.2.1',
    status: 'running',
    health: 'degraded',
    agentName: 'prod-agent-us-east-1',
    agentConnected: true,
    lastDeployedAt: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    lastHeartbeat: new Date(Date.now() - 3 * 60 * 1000).toISOString(),
    environment: 'Production',
  },
];

export const MOCK_ARTIFACTS: MockArtifact[] = [
  {
    id: 'art-1',
    name: 'datastack/platform',
    type: 'container-image',
    latestVersion: 'v2.5.0',
    totalVersions: 45,
    totalDownloads: 1247,
    lastPushedAt: new Date(Date.now() - 35 * 24 * 60 * 60 * 1000).toISOString(),
    size: 524288000,
  },
  {
    id: 'art-2',
    name: 'datastack/analytics-engine',
    type: 'container-image',
    latestVersion: 'v1.9.0',
    totalVersions: 32,
    totalDownloads: 892,
    lastPushedAt: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000).toISOString(),
    size: 387096576,
  },
  {
    id: 'art-3',
    name: 'datastack/monitoring',
    type: 'helm-chart',
    latestVersion: 'v3.2.1',
    totalVersions: 18,
    totalDownloads: 543,
    lastPushedAt: new Date(Date.now() - 42 * 24 * 60 * 60 * 1000).toISOString(),
    size: 12582912,
  },
  {
    id: 'art-4',
    name: 'datastack/cli-tools',
    type: 'generic',
    latestVersion: 'v0.7.2',
    totalVersions: 12,
    totalDownloads: 234,
    lastPushedAt: new Date(Date.now() - 20 * 24 * 60 * 60 * 1000).toISOString(),
    size: 8388608,
  },
];
