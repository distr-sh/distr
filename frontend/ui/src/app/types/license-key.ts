import {BaseModel, Named} from '@distr-sh/distr-sdk';

export interface LicenseKey extends BaseModel, Named {
  description?: string;
  payload: Record<string, unknown>;
  notBefore: string;
  expiresAt: string;
  customerOrganizationId?: string;
}
