import {DatePipe} from '@angular/common';
import {Component, inject, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleInfo, faExclamationTriangle} from '@fortawesome/free-solid-svg-icons';
import {WEBSITE_URL} from '../../../../constants';
import {AuthService} from '../../../services/auth.service';
import {Organization} from '../../../types/organization';

@Component({
  selector: 'app-nav-bar-subscription-banner',
  imports: [FaIconComponent, DatePipe, RouterLink],
  templateUrl: './nav-bar-subscription-banner.component.html',
})
export class NavBarSubscriptionBannerComponent {
  private readonly auth = inject(AuthService);

  organization = input.required<Organization>();
  isTrial = input.required<boolean>();
  isSubscriptionExpired = input.required<boolean>();

  protected readonly faExclamationTriangle = faExclamationTriangle;
  protected readonly faCircleInfo = faCircleInfo;
  protected readonly websiteUrl = WEBSITE_URL;

  isVendorAdmin(): boolean {
    return this.auth.isVendor() && this.auth.hasRole('admin');
  }

  isVendorNonAdmin(): boolean {
    return this.auth.isVendor() && !this.auth.hasRole('admin');
  }
}
