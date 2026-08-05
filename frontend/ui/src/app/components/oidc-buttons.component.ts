import {ChangeDetectionStrategy, Component, inject, input} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGithub, faGoogle, faMicrosoft} from '@fortawesome/free-brands-svg-icons';
import {faArrowRightToBracket} from '@fortawesome/free-solid-svg-icons';
import {PortalService} from '../services/portal.service';

/**
 * Renders a button per instance-scoped OIDC provider offered on this host. The providers are configured per
 * instance, but are not offered on an organization's custom app domain, so the list is host-resolved.
 */
@Component({
  selector: 'app-oidc-buttons',
  imports: [FaIconComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './oidc-buttons.component.html',
})
export class OidcButtonsComponent {
  protected readonly loginConfig = inject(PortalService).loginConfig;

  readonly label = input('Or use one of these to sign in:');

  protected getLoginURL(provider: string): string {
    return `/api/v1/auth/oidc/${provider}`;
  }

  protected readonly faGithub = faGithub;
  protected readonly faGoogle = faGoogle;
  protected readonly faMicrosoft = faMicrosoft;
  protected readonly faArrowRightToBracket = faArrowRightToBracket;
}
