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
 * Whether a customer's license needs attention. A missing license only does once some customer has one, since until
 * then the vendor does not use licensing. Any license key or entitlement counts as a license, so this reflects whether
 * the customer has been set up rather than what it may access.
 */
export function isCustomerLicenseIssue(status: CustomerLicenseStatus | undefined, isLicensingInUse: boolean): boolean {
  return status === 'expired' || status === 'expiring' || (status === 'no_license' && isLicensingInUse);
}
