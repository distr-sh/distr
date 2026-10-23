import {Component, computed, inject, input, output} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faShuffle} from '@fortawesome/free-solid-svg-icons';
import {membershipLabel, OrganizationKind} from '../../../../util/organization-kind';
import {UserRoleLabelPipe} from '../../../../util/user-role';
import {AuthService} from '../../../services/auth.service';
import {OrganizationWithUserRole} from '../../../types/organization';

@Component({
  selector: 'app-nav-bar-lens-banner',
  imports: [FaIconComponent, UserRoleLabelPipe],
  templateUrl: './nav-bar-lens-banner.component.html',
})
export class NavBarLensBannerComponent {
  protected readonly isSuperAdmin = inject(AuthService).isSuperAdmin();

  public readonly organization = input.required<OrganizationWithUserRole>();
  public readonly ownKind = input.required<OrganizationKind>();
  public readonly leaveLens = output();

  protected readonly lensName = computed(() => membershipLabel(this.organization()));

  protected readonly faShuffle = faShuffle;
}
