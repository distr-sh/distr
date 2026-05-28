import {Component, inject, input} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGithub, faGoogle, faMicrosoft} from '@fortawesome/free-brands-svg-icons';
import {faArrowRightToBracket} from '@fortawesome/free-solid-svg-icons';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-oidc-buttons',
  imports: [FaIconComponent],
  templateUrl: './oidc-buttons.component.html',
})
export class OidcButtonsComponent {
  private readonly auth = inject(AuthService);
  protected readonly loginConfig = toSignal(this.auth.loginConfig$);

  readonly label = input('Or use one of these to sign in:');

  protected getLoginURL(provider: string): string {
    return `/api/v1/auth/oidc/${provider}`;
  }

  protected readonly faGithub = faGithub;
  protected readonly faGoogle = faGoogle;
  protected readonly faMicrosoft = faMicrosoft;
  protected readonly faArrowRightToBracket = faArrowRightToBracket;
}
