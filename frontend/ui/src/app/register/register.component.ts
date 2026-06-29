import {ChangeDetectionStrategy, Component, inject, OnInit} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {firstValueFrom} from 'rxjs';
import {PLACEHOLDER_EMAIL, PLACEHOLDER_NAME} from '../../constants';
import {getFormDisplayedError} from '../../util/errors';
import {OidcButtonsComponent} from '../components/oidc-buttons.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';

@Component({
  selector: 'app-register',
  imports: [RouterLink, ReactiveFormsModule, AutotrimDirective, OidcButtonsComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './register.component.html',
})
export class RegisterComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly auth = inject(AuthService);

  errorMessage?: string;
  loading = false;
  protected readonly placeholderName = PLACEHOLDER_NAME;
  protected readonly placeholderEmail = PLACEHOLDER_EMAIL;
  public readonly form = new FormGroup(
    {
      email: new FormControl('', [Validators.required, Validators.email]),
      name: new FormControl<string | undefined>(undefined),
      organizationName: new FormControl<string | undefined>(undefined),
      password: new FormControl('', [Validators.required, Validators.minLength(8)]),
      passwordConfirm: new FormControl('', [Validators.required]),
    },
    (control) => (control.value.password === control.value.passwordConfirm ? null : {passwordMismatch: 'error'})
  );

  ngOnInit() {
    const email = this.route.snapshot.queryParamMap.get('email');
    if (email) {
      this.form.patchValue({email});
    }
  }

  public async submit(): Promise<void> {
    this.form.markAllAsTouched();
    this.errorMessage = undefined;
    if (this.form.valid) {
      this.loading = true;
      const value = this.form.value;
      try {
        await firstValueFrom(this.auth.register(value.email!, value.name, value.organizationName, value.password!));
        location.assign('/');
      } catch (e) {
        this.errorMessage = getFormDisplayedError(e);
        this.loading = false;
      }
    }
  }
}
