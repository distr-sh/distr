/**
 * customerScopeParams builds the query that scopes a request to one customer. A vendor and a
 * partner send it to read and write what a customer of theirs has configured.
 */
export function customerScopeParams(customerOrganizationId?: string): Record<string, string> {
  return customerOrganizationId ? {customerOrganizationId} : {};
}
