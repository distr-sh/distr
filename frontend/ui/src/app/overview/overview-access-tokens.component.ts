import {DatePipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCode} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {RelativeDatePipe} from '../../util/dates';
import {OverviewAccessToken} from '../types/overview';

@Component({
  selector: 'app-overview-access-tokens',
  templateUrl: './overview-access-tokens.component.html',
  imports: [DatePipe, RouterLink, FaIconComponent, RelativeDatePipe],
})
export class OverviewAccessTokensComponent {
  public readonly accessTokens = input.required<OverviewAccessToken[]>();

  protected readonly faCode = faCode;

  protected readonly tokens = computed(() =>
    this.accessTokens().map((token) => ({...token, expired: dayjs(token.expiresAt).isBefore()}))
  );
}
