import {DatePipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faKey} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {RelativeDatePipe} from '../../../util/dates';
import {AlertComponent} from '../../components/alert/alert.component';
import {CustomerHomeLicense} from '../../types/customer-home';

const expiringSoonDays = 30;

type LicenseState =
  | {type: 'active'}
  | {type: 'not_yet_valid'; name: string; notBefore: string}
  | {type: 'valid_until' | 'expiring' | 'expired'; name: string; expiresAt: string};

@Component({
  selector: 'app-customer-home-license',
  templateUrl: './customer-home-license.component.html',
  imports: [DatePipe, RouterLink, FaIconComponent, RelativeDatePipe, AlertComponent],
})
export class CustomerHomeLicenseComponent {
  public readonly license = input.required<CustomerHomeLicense>();

  protected readonly faKey = faKey;

  protected readonly state = computed<LicenseState>(() => {
    const {name, notBefore, expiresAt} = this.license();
    if (notBefore !== undefined && dayjs(notBefore).isAfter()) {
      return {type: 'not_yet_valid', name, notBefore};
    }
    if (expiresAt === undefined) {
      return {type: 'active'};
    }
    const expires = dayjs(expiresAt);
    return {
      type: expires.isBefore()
        ? 'expired'
        : expires.isBefore(dayjs().add(expiringSoonDays, 'day'))
          ? 'expiring'
          : 'valid_until',
      name,
      expiresAt,
    };
  });
}
