import {Component, inject} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {firstValueFrom, lastValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AuthService} from '../services/auth.service';
import {SettingsService} from '../services/settings.service';

@Component({
  selector: 'app-password-reset',
  imports: [ReactiveFormsModule],
  templateUrl: './password-reset.component.html',
})
export class PasswordResetComponent {
  private readonly settings = inject(SettingsService);
  private readonly auth = inject(AuthService);

  public readonly form = new FormGroup(
    {
      password: new FormControl('', [Validators.required, Validators.minLength(8)]),
      passwordConfirm: new FormControl('', [Validators.required]),
    },
    (control) => (control.value.password === control.value.passwordConfirm ? null : {passwordMismatch: 'error'})
  );
  public readonly email = this.auth.getClaims()?.email;
  public errorMessage?: string;
  loading = false;

  public async submit() {
    this.form.markAllAsTouched();
    this.errorMessage = undefined;
    if (this.form.valid) {
      this.loading = true;
      try {
        await lastValueFrom(this.settings.updateUserSettings({password: this.form.value.password!}));
        await lastValueFrom(this.auth.logout());
        location.assign(`/login?email=${encodeURIComponent(this.email ?? '')}&reason=password-reset`);
      } catch (e) {
        this.errorMessage = getFormDisplayedError(e);
      } finally {
        this.loading = false;
      }
    }
  }

  public async logoutAndRedirectToLogin() {
    await firstValueFrom(this.auth.logout());
    location.assign('/login');
  }
}
