import {Pipe, PipeTransform} from '@angular/core';
import {CustomerOrganization} from '@distr-sh/distr-sdk';

export type OrganizationKind = 'vendor' | 'customer' | 'partner';

export interface OrganizationMembership {
  customerOrganizationId?: string;
  customerOrganization?: CustomerOrganization | undefined;
  partnerOrganizationId?: string;
}

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
