export interface DeploymentTargetLatestMetrics {
  deploymentTargetId: string;
  createdAt: string;
  cpuCoresMillis: number;
  cpuUsage: number;
  memoryBytes: number;
  memoryUsage: number;
  diskMetrics?: DeploymentTargetDiskMetric[];
}

interface DeploymentTargetDiskMetric {
  device: string;
  path: string;
  fsType: string;
  bytesTotal: number;
  bytesUsed: number;
}
