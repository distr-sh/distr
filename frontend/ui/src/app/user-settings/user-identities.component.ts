import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faGithub, faGoogle, faMicrosoft} from '@fortawesome/free-brands-svg-icons';
import {faArrowRightToBracket, faCircleExclamation, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {OIDCIdentity, OIDCProvider, SettingsService} from '../services/settings.service';
import {ToastService} from '../services/toast.service';

const oidcProviderNames: Record<OIDCProvider, string> = {
  github: 'GitHub',
  google: 'Google',
  microsoft: 'Microsoft',
  generic: 'OIDC Provider',
  custom: 'Organization provider',
};

const oidcProviderIcons: Record<OIDCProvider, IconDefinition> = {
  github: faGithub,
  google: faGoogle,
  microsoft: faMicrosoft,
  generic: faArrowRightToBracket,
  custom: faArrowRightToBracket,
};

@Component({
  selector: 'app-user-identities',
  templateUrl: './user-identities.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, DatePipe],
})
export class UserIdentitiesComponent {
  protected readonly faXmark = faXmark;
  protected readonly faCircleExclamation = faCircleExclamation;

  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly settingsService = inject(SettingsService);
  private readonly overlay = inject(OverlayService);

  protected readonly customOidcSession = this.auth.isCustomOidcSession();

  protected readonly formLoading = signal(false);
  protected readonly oidcIdentities = signal<OIDCIdentity[]>([]);
  protected readonly oidcIdentityToDisconnect = signal<OIDCIdentity | undefined>(undefined);
  protected disconnectOidcIdentityDialogRef?: DialogRef<void>;

  private readonly disconnectOidcIdentityDialog =
    viewChild.required<TemplateRef<unknown>>('disconnectOidcIdentityDialog');

  constructor() {
    this.loadOidcIdentities();
  }

  protected providerName(identity: OIDCIdentity): string {
    return identity.configurationName ?? oidcProviderNames[identity.provider] ?? identity.provider;
  }

  protected providerIcon(provider: OIDCProvider): IconDefinition {
    return oidcProviderIcons[provider] ?? faArrowRightToBracket;
  }

  protected showDisconnectOidcIdentityDialog(identity: OIDCIdentity): void {
    this.oidcIdentityToDisconnect.set(identity);
    this.disconnectOidcIdentityDialogRef?.dismiss();
    this.disconnectOidcIdentityDialogRef = this.overlay.showModal<void>(this.disconnectOidcIdentityDialog());
  }

  protected async disconnectOidcIdentity(): Promise<void> {
    const identity = this.oidcIdentityToDisconnect();
    if (!identity) {
      return;
    }

    try {
      this.formLoading.set(true);
      await firstValueFrom(this.settingsService.deleteOIDCIdentity(identity.id));
      this.toast.success(`${this.providerName(identity)} disconnected successfully.`);
      this.disconnectOidcIdentityDialogRef?.close();
      this.oidcIdentityToDisconnect.set(undefined);
      await this.loadOidcIdentities();
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  private async loadOidcIdentities(): Promise<void> {
    try {
      this.oidcIdentities.set(await firstValueFrom(this.settingsService.getOIDCIdentities()));
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    }
  }
}
