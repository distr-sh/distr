import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {firstValueFrom} from 'rxjs';
import {PLACEHOLDER_NAME} from '../../constants';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-invite',
  imports: [ReactiveFormsModule, AutotrimDirective],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './invite.component.html',
})
export class InviteComponent {
  private readonly auth = inject(AuthService);
  private readonly claims = this.auth.getClaims();
  public readonly email = this.claims?.email;
  protected readonly placeholderName = PLACEHOLDER_NAME;

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
      try {
        const value = this.form.value;
        await firstValueFrom(this.auth.acceptInvite(value.name, value.password!));
        location.assign('/');
      } catch (e) {
        this.errorMessage = getFormDisplayedError(e);
        this.submitted = false;
      }
    }
  }
}
