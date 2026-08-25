import {ChangeDetectionStrategy, Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faCircleExclamation, faExclamationTriangle, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {MFASetupData, SettingsService} from '../services/settings.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-user-security-settings',
  templateUrl: './user-security-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective],
})
export class UserSecuritySettingsComponent {
  protected readonly faCheck = faCheck;
  protected readonly faXmark = faXmark;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faExclamationTriangle = faExclamationTriangle;

  private readonly fb = inject(FormBuilder);
  private readonly ctx = inject(ContextService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly settingsService = inject(SettingsService);
  private readonly overlay = inject(OverlayService);

  protected readonly user = toSignal(this.ctx.getUser());

  protected readonly customOidcSession = this.auth.isCustomOidcSession();

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

  protected readonly formLoading = signal(false);
  protected readonly passwordResetLoading = signal(false);
  protected readonly passwordResetSent = signal(false);
  protected readonly mfaSetupData = signal<MFASetupData | undefined>(undefined);
  protected readonly recoveryCodes = signal<string[] | undefined>(undefined);
  protected readonly recoveryCodesCount = signal<number>(0);
  protected disableMfaDialogRef?: DialogRef<void>;
  protected regenerateRecoveryCodesDialogRef?: DialogRef<void>;

  private readonly disableMfaDialog = viewChild.required<TemplateRef<unknown>>('disableMfaDialog');
  private readonly regenerateRecoveryCodesDialog = viewChild.required<TemplateRef<unknown>>(
    'regenerateRecoveryCodesDialog'
  );

  constructor() {
    this.loadRecoveryCodesStatus();
  }

  protected async requestPasswordReset(): Promise<void> {
    const email = this.user()?.email;
    if (!email) {
      return;
    }

    try {
      this.passwordResetLoading.set(true);
      await firstValueFrom(this.auth.resetPassword(email));
      this.passwordResetSent.set(true);
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.passwordResetLoading.set(false);
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

  protected disableMfa() {
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

  protected showRegenerateRecoveryCodesDialog() {
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

  private async loadRecoveryCodesStatus(): Promise<void> {
    const user = await firstValueFrom(this.ctx.getUser());
    if (user.mfaEnabled) {
      try {
        const status = await firstValueFrom(this.settingsService.getMFARecoveryCodesStatus());
        this.recoveryCodesCount.set(status.remainingCodes);
      } catch (e) {
        // Silently fail, not critical
      }
    }
  }
}
