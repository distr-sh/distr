import {DatePipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCode} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {RelativeDatePipe} from '../../../util/dates';
import {CustomerHomeAccessToken} from '../../types/customer-home';

@Component({
  selector: 'app-customer-home-access-tokens',
  templateUrl: './customer-home-access-tokens.component.html',
  imports: [DatePipe, RouterLink, FaIconComponent, RelativeDatePipe],
})
export class CustomerHomeAccessTokensComponent {
  public readonly accessTokens = input.required<CustomerHomeAccessToken[]>();

  protected readonly faCode = faCode;

  protected readonly tokens = computed(() =>
    this.accessTokens().map((token) => ({...token, expired: dayjs(token.expiresAt).isBefore()}))
  );
}
