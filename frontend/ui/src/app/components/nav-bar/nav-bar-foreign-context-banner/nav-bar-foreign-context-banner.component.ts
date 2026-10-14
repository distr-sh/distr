import {Component, computed, inject, input, output} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faShuffle} from '@fortawesome/free-solid-svg-icons';
import {membershipLabel, OrganizationKind, organizationKind} from '../../../../util/organization-kind';
import {UserRoleLabelPipe} from '../../../../util/user-role';
import {AuthService} from '../../../services/auth.service';
import {OrganizationWithUserRole} from '../../../types/organization';

@Component({
  selector: 'app-nav-bar-foreign-context-banner',
  imports: [FaIconComponent, UserRoleLabelPipe],
  templateUrl: './nav-bar-foreign-context-banner.component.html',
})
export class NavBarForeignContextBannerComponent {
  protected readonly isSuperAdmin = inject(AuthService).isSuperAdmin();

  public readonly organization = input.required<OrganizationWithUserRole>();
  public readonly ownPortalKind = input.required<OrganizationKind>();
  public readonly returnToOwnPortal = output();

  protected readonly portalKind = computed(() => organizationKind(this.organization()));
  protected readonly portalName = computed(() => membershipLabel(this.organization()));
  protected readonly isCustomerLens = computed(() => !this.isSuperAdmin && this.portalKind() === 'customer');

  protected readonly faShuffle = faShuffle;
}
