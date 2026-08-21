export interface DeploymentTargetLatestMetrics {
  deploymentTargetId: string;
  cpuCoresMillis: number;
  cpuUsage: number;
  memoryBytes: number;
  memoryUsage: number;
  agentCpuUsageMillis?: number;
  agentMemoryBytes?: number;
  diskMetrics?: DeploymentTargetDiskMetric[];
}

export interface DeploymentResourceMetrics {
  deploymentId: string;
  createdAt: string;
  resources: DeploymentResourceMetric[];
}

export interface DeploymentResourceMetric {
  resource: string;
  container: string;
  cpuUsageMillis: number;
  memoryBytes: number;
  cpuLimitMillis?: number;
  memoryLimitBytes?: number;
}

interface DeploymentTargetDiskMetric {
  device: string;
  path: string;
  fsType: string;
  bytesTotal: number;
  bytesUsed: number;
}
