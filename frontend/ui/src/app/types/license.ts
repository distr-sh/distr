import {CustomerOrganization} from '@distr-sh/distr-sdk';
import {ApplicationEntitlement} from './application-entitlement';
import {ArtifactEntitlement} from './artifact-entitlement';
import {LicenseKey} from './license-key';

export interface License {
  customerOrganization: CustomerOrganization;
  applicationEntitlements: ApplicationEntitlement[];
  artifactEntitlements: ArtifactEntitlement[];
  licenseKeys: LicenseKey[];
}
