import {Pipe, PipeTransform} from '@angular/core';

export type OrganizationKind = 'vendor' | 'customer' | 'partner';

export interface OrganizationMembership {
  customerOrganizationId?: string;
  partnerOrganizationId?: string;
}

export function organizationKind(membership: OrganizationMembership): OrganizationKind {
  if (membership.partnerOrganizationId) {
    return 'partner';
  }
  if (membership.customerOrganizationId) {
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
