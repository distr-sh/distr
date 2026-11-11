import {Component, inject, input, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowRightLong, faXmark} from '@fortawesome/free-solid-svg-icons';
import {WEBSITE_URL} from '../../../constants';
import {buildConfig} from '../../../data';
import {environment} from '../../../env/env';
import {RequireVendorDirective} from '../../directives/required-role.directive';
import {NavigationService} from '../../navigation/navigation.service';
import {AuthService} from '../../services/auth.service';
import {OrganizationService} from '../../services/organization.service';
import {SidebarService} from '../../services/sidebar.service';
import {SideBarItemComponent} from './side-bar-item.component';

@Component({
  selector: 'app-side-bar',
  templateUrl: './side-bar.component.html',
  imports: [FaIconComponent, RequireVendorDirective, SideBarItemComponent],
})
export class SideBarComponent {
  protected readonly auth = inject(AuthService);
  protected readonly sidebar = inject(SidebarService);
  protected readonly navigation = inject(NavigationService);
  private readonly organizationService = inject(OrganizationService);

  protected readonly buildConfig = buildConfig;
  protected readonly edition = environment.edition;
  protected readonly websiteUrl = WEBSITE_URL;

  protected readonly faArrowRightLong = faArrowRightLong;
  protected readonly faXmark = faXmark;

  protected readonly isFeedbackAlertVisible = signal(true);

  public readonly isBannerVisible = input<boolean>();
  public readonly isSidebarVisible = input<boolean>();

  protected readonly hasSubscription = this.organizationService.hasSubscription;
}
