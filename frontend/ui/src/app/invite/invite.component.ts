import {ChangeDetectionStrategy, Component, inject, signal} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {MfaCodeInputComponent, newMfaCodeControl} from '../components/mfa-code-input.component';
import {PortalLogoComponent} from '../components/portal-logo/portal-logo.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {PlaceholderDirective} from '../directives/placeholder.directive';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-invite',
  imports: [ReactiveFormsModule, AutotrimDirective, PlaceholderDirective, PortalLogoComponent, MfaCodeInputComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './invite.component.html',
})
export class InviteComponent {
  private readonly auth = inject(AuthService);
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
  public readonly mfaCode = newMfaCodeControl();
  public readonly mfaRequired = signal(false);
  public readonly submitted = signal(false);
  public readonly errorMessage = signal<string | undefined>(undefined);

  public async submit(): Promise<void> {
    this.form.markAllAsTouched();
    this.errorMessage.set(undefined);
    if (this.mfaRequired()) {
      this.mfaCode.markAsTouched();
      if (this.mfaCode.invalid) {
        return;
      }
    }
    if (this.form.valid) {
      this.submitted.set(true);
      try {
        const value = this.form.value;
        const {requiresMfa} = await firstValueFrom(
          this.auth.acceptInvite(value.name, value.password!, this.mfaCode.value || undefined)
        );
        if (requiresMfa) {
          this.mfaRequired.set(true);
          this.submitted.set(false);
        } else {
          location.assign('/');
        }
      } catch (e) {
        this.errorMessage.set(getFormDisplayedError(e));
        this.submitted.set(false);
      }
    }
  }
}
