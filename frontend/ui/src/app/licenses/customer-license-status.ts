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

/**
 * Whether a customer's license needs attention. A missing license only does once some customer has one, because
 * until then every customer sees all applications and artifacts.
 */
export function isCustomerLicenseIssue(status: CustomerLicenseStatus | undefined, isLicensingInUse: boolean): boolean {
  return status === 'expired' || status === 'expiring' || (status === 'no_license' && isLicensingInUse);
}
