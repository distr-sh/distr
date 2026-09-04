import {Pipe, PipeTransform} from '@angular/core';
import {CustomerOrganization} from '@distr-sh/distr-sdk';

export type OrganizationKind = 'vendor' | 'customer' | 'partner';

export interface OrganizationMembership {
  customerOrganizationId?: string;
  customerOrganization?: CustomerOrganization | undefined;
  partnerOrganizationId?: string;
}

export interface NamedOrganizationMembership extends OrganizationMembership {
  customerOrganizationName?: string;
  partnerOrganizationName?: string;
}

const ORGANIZATION_KIND_LABELS: Record<OrganizationKind, string> = {
  vendor: 'Vendor',
  customer: 'Customer',
  partner: 'Partner',
};

export function organizationKind(membership: OrganizationMembership): OrganizationKind {
  if (membership.partnerOrganizationId) {
    return 'partner';
  }
  if (membership.customerOrganizationId || membership.customerOrganization?.id) {
    return 'customer';
  }
  return 'vendor';
}

@Pipe({name: 'organizationKind'})
export class OrganizationKindPipe implements PipeTransform {
  transform(membership: OrganizationMembership): OrganizationKind {
    return organizationKind(membership);
  }
}

// a customer or partner membership is named after the organization it belongs to, which tells the
// user more than the words "customer" and "partner" do
export function membershipLabel(membership: NamedOrganizationMembership): string {
  const kind = organizationKind(membership);
  if (kind === 'customer' && membership.customerOrganizationName) {
    return membership.customerOrganizationName;
  }
  if (kind === 'partner' && membership.partnerOrganizationName) {
    return membership.partnerOrganizationName;
  }
  return ORGANIZATION_KIND_LABELS[kind];
}

@Pipe({name: 'membershipLabel'})
export class MembershipLabelPipe implements PipeTransform {
  transform(membership: NamedOrganizationMembership): string {
    return membershipLabel(membership);
  }
}
