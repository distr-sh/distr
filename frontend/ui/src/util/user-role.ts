import {Pipe, PipeTransform} from '@angular/core';
import {UserRole} from '@distr-sh/distr-sdk';
import {USER_ROLE_LABELS} from '../app/components/user-role-select.component';

@Pipe({name: 'userRoleLabel'})
export class UserRoleLabelPipe implements PipeTransform {
  transform(value: UserRole | null | undefined): string {
    return value ? USER_ROLE_LABELS[value] : '';
  }
}
