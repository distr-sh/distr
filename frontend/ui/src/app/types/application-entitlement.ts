import {Application, ApplicationVersion, BaseModel, CustomerOrganization, Named} from '@distr-sh/distr-sdk';

export interface ApplicationEntitlement extends BaseModel, Named {
  expiresAt?: Date;
  applicationId?: string;
  application?: Application;
  versions?: ApplicationVersion[];
  customerOrganizationId?: string;
  customerOrganization?: CustomerOrganization;

  registryUrl?: string;
  registryUsername?: string;
  registryPassword?: string;
}
