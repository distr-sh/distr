import {Component, inject} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {SettingsService} from '../services/settings.service';

@Component({
  selector: 'app-invite',
  imports: [ReactiveFormsModule, AutotrimDirective],
  templateUrl: './invite.component.html',
})
export class InviteComponent {
  private readonly auth = inject(AuthService);
  private readonly settings = inject(SettingsService);
  private readonly claims = this.auth.getClaims();
  public readonly email = this.claims?.email;

  public readonly form = new FormGroup(
    {
      name: new FormControl<string | undefined>(this.claims?.name, {nonNullable: true}),
      password: new FormControl('', {nonNullable: true, validators: [Validators.required, Validators.minLength(8)]}),
      passwordConfirm: new FormControl('', [Validators.required]),
    },
    (control) => (control.value.password === control.value.passwordConfirm ? null : {passwordMismatch: 'error'})
  );
  public submitted = false;
  errorMessage?: string;

  public async submit(): Promise<void> {
    this.form.markAllAsTouched();
    this.errorMessage = undefined;
    if (this.form.valid) {
      this.submitted = true;
      let updateOk = false;
      try {
        const value = this.form.value;
        await firstValueFrom(this.settings.updateUserSettings({name: value.name, password: value.password}));
        updateOk = true;
      } catch (e) {
        this.errorMessage = getFormDisplayedError(e);
      } finally {
        this.submitted = false;
      }

      if (updateOk) {
        try {
          if (this.claims?.email_verified) {
            await firstValueFrom(this.auth.confirmEmailVerification());
          }
        } catch (e) {
          // ignore errors of confirmation (because password has already been set)
        } finally {
          this.auth.logout();
          location.assign(`/login?email=${encodeURIComponent(this.email ?? '')}&inviteSuccess=true`);
        }
      }
    }
  }
}
