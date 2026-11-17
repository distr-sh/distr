import {never} from '../../../util/exhaust';
import {BadgeVariant} from '../badge';
import {CustomerLicenseStatus} from './customer-list-mappers';

export function customerLicenseStatusVariant(status: CustomerLicenseStatus): BadgeVariant {
  switch (status) {
    case 'no_license':
    case 'expiring':
      return 'warning';
    case 'expired':
      return 'error';
    case 'valid':
      return 'success';
    default:
      return never(status);
  }
}
