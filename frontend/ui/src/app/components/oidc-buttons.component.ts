import {ChangeDetectionStrategy, Component, inject, input} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGithub, faGoogle, faMicrosoft} from '@fortawesome/free-brands-svg-icons';
import {faArrowRightToBracket} from '@fortawesome/free-solid-svg-icons';
import {PortalService} from '../services/portal.service';

/**
 * Renders a button per OIDC provider offered on this host: the instance-scoped providers on the default host, and
 * the providers configured by an organization on its own custom app domain. The two sets never appear together,
 * because the list is host-resolved.
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

  protected getCustomLoginURL(configurationId: string): string {
    return `/api/v1/auth/oidc/custom/${configurationId}`;
  }

  protected readonly faGithub = faGithub;
  protected readonly faGoogle = faGoogle;
  protected readonly faMicrosoft = faMicrosoft;
  protected readonly faArrowRightToBracket = faArrowRightToBracket;
}
