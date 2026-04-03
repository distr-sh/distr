import {BaseModel, Named} from '@distr-sh/distr-sdk';

export interface LicenseKey extends BaseModel, Named {
  description?: string;
  payload: Record<string, unknown>;
  notBefore: string;
  expiresAt: string;
  lastRevisedAt?: string;
  customerOrganizationId?: string;
}

export interface LicenseKeyRevision extends BaseModel {
  notBefore: string;
  expiresAt: string;
  payload: Record<string, unknown>;
}
