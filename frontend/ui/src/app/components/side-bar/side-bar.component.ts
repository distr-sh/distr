import {Component, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowRightLong, faXmark} from '@fortawesome/free-solid-svg-icons';
import {map} from 'rxjs';
import {WEBSITE_URL} from '../../../constants';
import {buildConfig} from '../../../data';
import {environment} from '../../../env/env';
import {RequireVendorDirective} from '../../directives/required-role.directive';
import {NavigationService} from '../../navigation/navigation.service';
import {AuthService} from '../../services/auth.service';
import {OrganizationService} from '../../services/organization.service';
import {SidebarService} from '../../services/sidebar.service';
import {isPayingSubscription} from '../../types/subscription';
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

  protected readonly isAnnouncementVisible = signal(true);

  public readonly isBannerVisible = input<boolean>();
  public readonly isSidebarVisible = input<boolean>();

  protected readonly hasSubscription = toSignal(
    this.organizationService.get().pipe(map((org) => isPayingSubscription(org.subscriptionType))),
    {initialValue: false}
  );
}
