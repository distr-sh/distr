import {Pipe, PipeTransform} from '@angular/core';
import {UserRole} from '@distr-sh/distr-sdk';

export const ALL_USER_ROLES: UserRole[] = ['read_only', 'read_write', 'admin'];

export const USER_ROLE_LABELS: Record<UserRole, string> = {
  read_only: 'Viewer',
  read_write: 'User',
  admin: 'Administrator',
};

export function userRolesAtOrBelow(max: UserRole | undefined): UserRole[] {
  if (!max) {
    return ALL_USER_ROLES;
  }
  // ALL_USER_ROLES is ordered low → high, so the index is the rank.
  return ALL_USER_ROLES.slice(0, ALL_USER_ROLES.indexOf(max) + 1);
}

@Pipe({name: 'userRoleLabel'})
export class UserRoleLabelPipe implements PipeTransform {
  transform(value: UserRole | null | undefined): string {
    return value ? USER_ROLE_LABELS[value] : '';
  }
}
