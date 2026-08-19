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

export interface DeploymentWorkloadMetrics {
  deploymentId: string;
  createdAt: string;
  workloads: DeploymentWorkloadMetric[];
}

export interface DeploymentWorkloadMetric {
  workload: string;
  name: string;
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
