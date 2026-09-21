import {ChangeDetectionStrategy, Component, inject, signal} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {MfaCodeInputComponent, newMfaCodeControl} from '../components/mfa-code-input.component';
import {PortalLogoComponent} from '../components/portal-logo/portal-logo.component';
import {PlaceholderDirective} from '../directives/placeholder.directive';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-password-reset',
  imports: [ReactiveFormsModule, PlaceholderDirective, PortalLogoComponent, MfaCodeInputComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './password-reset.component.html',
})
export class PasswordResetComponent {
  private readonly auth = inject(AuthService);

  public readonly form = new FormGroup(
    {
      password: new FormControl('', [Validators.required, Validators.minLength(8)]),
      passwordConfirm: new FormControl('', [Validators.required]),
    },
    (control) => (control.value.password === control.value.passwordConfirm ? null : {passwordMismatch: 'error'})
  );
  public readonly mfaCode = newMfaCodeControl();
  public readonly mfaRequired = signal(false);
  public readonly email = this.auth.getClaims()?.email;
  public readonly errorMessage = signal<string | undefined>(undefined);
  public readonly loading = signal(false);

  public async submit() {
    this.form.markAllAsTouched();
    this.errorMessage.set(undefined);
    if (this.mfaRequired()) {
      this.mfaCode.markAsTouched();
      if (this.mfaCode.invalid) {
        return;
      }
    }
    if (this.form.valid) {
      this.loading.set(true);
      try {
        const {requiresMfa} = await firstValueFrom(
          this.auth.confirmPasswordReset(this.form.value.password!, this.mfaCode.value || undefined)
        );
        if (requiresMfa) {
          this.mfaRequired.set(true);
          this.loading.set(false);
        } else {
          location.assign('/');
        }
      } catch (e) {
        this.errorMessage.set(getFormDisplayedError(e));
        this.loading.set(false);
      }
    }
  }

  public async logoutAndRedirectToLogin() {
    await firstValueFrom(this.auth.logout());
    location.assign('/login');
  }
}
