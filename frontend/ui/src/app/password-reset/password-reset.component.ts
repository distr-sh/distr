import {Component, inject} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-password-reset',
  imports: [ReactiveFormsModule],
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
  public readonly email = this.auth.getClaims()?.email;
  public errorMessage?: string;
  loading = false;

  public async submit() {
    this.form.markAllAsTouched();
    this.errorMessage = undefined;
    if (this.form.valid) {
      this.loading = true;
      try {
        await firstValueFrom(this.auth.confirmPasswordReset(this.form.value.password!));
        location.assign('/');
      } catch (e) {
        this.errorMessage = getFormDisplayedError(e);
        this.loading = false;
      }
    }
  }

  public async logoutAndRedirectToLogin() {
    await firstValueFrom(this.auth.logout());
    location.assign('/login');
  }
}
