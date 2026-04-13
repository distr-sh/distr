import {BaseModel, Named} from '@distr-sh/distr-sdk';

export interface LicenseTemplate extends BaseModel, Named {
  payloadTemplate: string;
  expirationGracePeriodDays: number;
}
