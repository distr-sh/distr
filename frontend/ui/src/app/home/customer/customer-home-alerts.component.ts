import {NgPlural, NgPluralCase} from '@angular/common';
import {Component, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBell} from '@fortawesome/free-solid-svg-icons';
import {CustomerHomeAlerts} from '../../types/customer-home';

@Component({
  selector: 'app-customer-home-alerts',
  templateUrl: './customer-home-alerts.component.html',
  imports: [NgPlural, NgPluralCase, RouterLink, FaIconComponent],
})
export class CustomerHomeAlertsComponent {
  public readonly alerts = input.required<CustomerHomeAlerts>();
  public readonly canConfigure = input.required<boolean>();

  protected readonly faBell = faBell;
}
