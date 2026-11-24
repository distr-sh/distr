import {CustomerLicenseStatus} from '../components/customer-organizations/customer-list-mappers';
import {MultiSelectOption} from '../components/multi-select/multi-select.component';

export const customerLicenseStatusSelectOptions: MultiSelectOption<CustomerLicenseStatus>[] = [
  {value: 'valid', label: 'Valid'},
  {value: 'expiring', label: 'Expiring'},
  {value: 'expired', label: 'Expired'},
  {value: 'no_license', label: 'No license'},
];

export function isCustomerLicenseStatus(value: string): value is CustomerLicenseStatus {
  return customerLicenseStatusSelectOptions.some((option) => option.value === value);
}
