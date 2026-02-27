import {BaseModel, Named} from '@distr-sh/distr-sdk';

export interface UsageLicense extends BaseModel, Named {
  description?: string;
  payload: Record<string, unknown>;
  token: string;
  notBefore: string;
  expiresAt: string;
  customerOrganizationId?: string;
}
