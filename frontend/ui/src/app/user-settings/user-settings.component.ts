import {AsyncPipe, DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faGithub, faGoogle, faMicrosoft} from '@fortawesome/free-brands-svg-icons';
import {
  faArrowRightToBracket,
  faCheck,
  faCircleExclamation,
  faExclamationTriangle,
  faFloppyDisk,
  faKey,
  faPen,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {filter, firstValueFrom, take} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {SecureImagePipe} from '../../util/secureImage';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {ImageUploadService} from '../services/image-upload.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {MFASetupData, OIDCIdentity, OIDCProvider, SettingsService} from '../services/settings.service';
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
  templateUrl: './user-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, SecureImagePipe, AsyncPipe, DatePipe, RouterLink],
})
export class UserSettingsComponent {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faPen = faPen;
  protected readonly faKey = faKey;
  protected readonly faCheck = faCheck;
  protected readonly faXmark = faXmark;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faExclamationTriangle = faExclamationTriangle;

  private readonly fb = inject(FormBuilder);
  private readonly ctx = inject(ContextService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly settingsService = inject(SettingsService);
  private readonly overlay = inject(OverlayService);

  protected readonly user = toSignal(this.ctx.getUser());

  // A session cannot become one, or stop being one, without a new login, so this is read once.
  protected readonly customOidcSession = this.auth.isCustomOidcSession();

  protected readonly generalForm = this.fb.group({
    name: this.fb.control(''),
    imageId: this.fb.control(''),
  });

  protected readonly emailForm = this.fb.group({
    email: this.fb.control('', [Validators.required, Validators.email]),
  });

  protected readonly setupMfaForm = this.fb.group({
    mfaCode: this.fb.control('', [
      Validators.required,
      Validators.pattern(/^\d*$/),
      Validators.minLength(6),
      Validators.maxLength(6),
    ]),
  });

  protected readonly disableMfaForm = this.fb.group({
    password: this.fb.control('', [Validators.required]),
  });

  protected readonly regenerateRecoveryCodesForm = this.fb.group({
    password: this.fb.control('', [Validators.required]),
  });

  protected readonly formLoading = signal(true);
  protected readonly mfaSetupData = signal<MFASetupData | undefined>(undefined);
  protected readonly recoveryCodes = signal<string[] | undefined>(undefined);
  protected readonly recoveryCodesCount = signal<number>(0);
  protected readonly oidcIdentities = signal<OIDCIdentity[]>([]);
  protected disableMfaDialogRef?: DialogRef<void>;
  protected regenerateRecoveryCodesDialogRef?: DialogRef<void>;
  protected disconnectOidcIdentityDialogRef?: DialogRef<void>;
  protected readonly oidcIdentityToDisconnect = signal<OIDCIdentity | undefined>(undefined);

  private readonly disableMfaDialog = viewChild.required<TemplateRef<unknown>>('disableMfaDialog');
  private readonly regenerateRecoveryCodesDialog = viewChild.required<TemplateRef<unknown>>(
    'regenerateRecoveryCodesDialog'
  );
  private readonly disconnectOidcIdentityDialog =
    viewChild.required<TemplateRef<unknown>>('disconnectOidcIdentityDialog');

  constructor() {
    this.ctx
      .getUser()
      .pipe(take(1), takeUntilDestroyed())
      .subscribe((user) => {
        this.generalForm.patchValue(user);
        this.formLoading.set(false);
        this.loadRecoveryCodesStatus();
      });
    this.loadOidcIdentities();
  }

  protected async showProfilePictureDialog() {
    this.imageUploadService
      .showDialog({imageUrl: this.user()?.imageUrl, scope: 'platform', showSuccessNotification: false})
      .pipe(filter((id) => id !== null))
      .subscribe((imageId) => {
        this.generalForm.patchValue({imageId});
        this.generalForm.markAsDirty();
      });
  }

  protected async saveGeneral(): Promise<void> {
    if (this.generalForm.invalid) {
      this.generalForm.markAllAsTouched();
      return;
    }

    try {
      this.formLoading.set(true);
      const result = await firstValueFrom(
        this.settingsService.updateUserSettings({
          name: this.generalForm.value.name ?? undefined,
          imageId: this.generalForm.value.imageId || undefined,
        })
      );
      this.toast.success('User settings saved successfully.');
      this.generalForm.patchValue(result);
      this.generalForm.markAsPristine();
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  protected async saveEmail(): Promise<void> {
    const email = this.emailForm.value.email;
    if (this.emailForm.invalid || !email) {
      this.emailForm.markAllAsTouched();
      return;
    }

    try {
      this.formLoading.set(true);
      await firstValueFrom(this.settingsService.requestEmailVerification(email));
      this.toast.success('Verification request sent. Please check your inbox.');
      this.emailForm.reset();
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  protected async startMfaSetup(): Promise<void> {
    try {
      this.mfaSetupData.set(await firstValueFrom(this.settingsService.startMFASetup()));
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    }
  }

  protected async enableMfa(): Promise<void> {
    const code = this.setupMfaForm.value.mfaCode;
    if (this.setupMfaForm.invalid || !code) {
      this.setupMfaForm.markAllAsTouched();
      return;
    }

    try {
      this.formLoading.set(true);
      const response = await firstValueFrom(this.settingsService.enableMFA(code));
      this.recoveryCodes.set(response.recoveryCodes);
      this.toast.success('Multi-factor authentication enabled successfully.');
      this.mfaSetupData.set(undefined);
      this.setupMfaForm.reset();
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  protected async disableMfa() {
    this.disableMfaDialogRef?.dismiss();
    this.disableMfaDialogRef = this.overlay.showModal<void>(this.disableMfaDialog());
  }

  protected async disableMfaFinish() {
    const password = this.disableMfaForm.value.password;
    if (this.disableMfaForm.invalid || !password) {
      this.disableMfaForm.markAllAsTouched();
      return;
    }

    try {
      this.formLoading.set(true);
      await firstValueFrom(this.settingsService.disableMFA(password));
      this.toast.success('Multi-factor authentication disabled successfully.');
      this.disableMfaDialogRef?.close();
      this.disableMfaForm.reset();
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  protected acknowledgeRecoveryCodes(): void {
    this.recoveryCodes.set(undefined);
    this.loadRecoveryCodesStatus();
  }

  protected copyRecoveryCodes(): void {
    const codes = this.recoveryCodes();
    if (codes) {
      navigator.clipboard.writeText(codes.join('\n'));
      this.toast.success('Recovery codes copied to clipboard');
    }
  }

  protected downloadRecoveryCodes(): void {
    const codes = this.recoveryCodes();
    if (codes) {
      const blob = new Blob([codes.join('\n')], {type: 'text/plain'});
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'distr-recovery-codes.txt';
      a.click();
      URL.revokeObjectURL(url);
    }
  }

  protected async showRegenerateRecoveryCodesDialog() {
    this.regenerateRecoveryCodesDialogRef?.dismiss();
    this.regenerateRecoveryCodesDialogRef = this.overlay.showModal<void>(this.regenerateRecoveryCodesDialog());
  }

  protected async regenerateRecoveryCodes(): Promise<void> {
    const password = this.regenerateRecoveryCodesForm.value.password;
    if (this.regenerateRecoveryCodesForm.invalid || !password) {
      this.regenerateRecoveryCodesForm.markAllAsTouched();
      return;
    }

    try {
      this.formLoading.set(true);
      const response = await firstValueFrom(this.settingsService.regenerateMFARecoveryCodes(password));
      this.recoveryCodes.set(response.recoveryCodes);
      this.regenerateRecoveryCodesDialogRef?.close();
      this.regenerateRecoveryCodesForm.reset();
      this.toast.success('Recovery codes regenerated successfully.');
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
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

  protected async loadRecoveryCodesStatus(): Promise<void> {
    if (this.user()?.mfaEnabled) {
      try {
        const status = await firstValueFrom(this.settingsService.getMFARecoveryCodesStatus());
        this.recoveryCodesCount.set(status.remainingCodes);
      } catch (e) {
        // Silently fail, not critical
      }
    }
  }
}
