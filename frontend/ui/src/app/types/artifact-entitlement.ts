import {BaseModel, Named} from '@distr-sh/distr-sdk';

export interface ArtifactEntitlementSelection {
  artifactId: string;
  versionIds?: string[];
}

export interface ArtifactEntitlement extends BaseModel, Named {
  expiresAt?: Date;
  artifacts?: ArtifactEntitlementSelection[];
  customerOrganizationId?: string;
}
