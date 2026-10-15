import {NgPlural, NgPluralCase} from '@angular/common';
import {Component, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBell} from '@fortawesome/free-solid-svg-icons';
import {OverviewAlerts} from '../types/overview';

@Component({
  selector: 'app-overview-alerts',
  templateUrl: './overview-alerts.component.html',
  imports: [NgPlural, NgPluralCase, RouterLink, FaIconComponent],
})
export class OverviewAlertsComponent {
  public readonly alerts = input.required<OverviewAlerts>();
  public readonly canConfigure = input.required<boolean>();

  protected readonly faBell = faBell;
}
