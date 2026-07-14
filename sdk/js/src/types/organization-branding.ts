import {BaseModel} from './base';

export interface OrganizationBranding extends BaseModel {
  title?: string;
  description?: string;
  logoImageId?: string;
  appDomain?: string;
  registryDomain?: string;
  emailFromAddress?: string;
  pageTitle?: string;
  faviconImageId?: string;
}
