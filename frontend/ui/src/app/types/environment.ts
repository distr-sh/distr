export interface Environment {
  id: string;
  createdAt: string;
  updatedAt: string;
  name: string;
  description: string;
  sortOrder: number;
  isProduction: boolean;
  allowDynamicTargets: boolean;
  retentionPolicyId?: string;
}

export interface CreateUpdateEnvironmentRequest {
  name: string;
  description: string;
  sortOrder: number;
  isProduction?: boolean;
  allowDynamicTargets?: boolean;
  retentionPolicyId?: string;
}
