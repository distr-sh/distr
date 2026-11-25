import {never} from '../../../util/exhaust';
import {BadgeVariant} from '../badge';
import {MultiSelectOption} from '../multi-select/multi-select.component';
import {CustomerLicenseStatus} from './customer-list-mappers';

export type CustomerUpdateStatus = 'outdated' | 'up_to_date';

export const customerUpdateStatusSelectOptions: MultiSelectOption<CustomerUpdateStatus>[] = [
  {value: 'up_to_date', label: 'Up to date'},
  {value: 'outdated', label: 'Outdated'},
];

export function isCustomerUpdateStatus(value: string): value is CustomerUpdateStatus {
  return customerUpdateStatusSelectOptions.some((option) => option.value === value);
}

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
