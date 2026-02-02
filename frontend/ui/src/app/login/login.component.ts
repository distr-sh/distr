import {AsyncPipe} from '@angular/common';
import {Component, inject, OnInit, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGithub, faGoogle, faMicrosoft} from '@fortawesome/free-brands-svg-icons';
import {faArrowRightToBracket} from '@fortawesome/free-solid-svg-icons/faArrowRightToBracket';
import {distinctUntilChanged, filter, lastValueFrom, map, take} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-login',
  imports: [ReactiveFormsModule, RouterLink, AutotrimDirective, AsyncPipe, FaIconComponent],
  templateUrl: './login.component.html',
})
export class LoginComponent implements OnInit {
  private readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly emailPasswordForm = this.fb.group({
    email: this.fb.control('', [Validators.required, Validators.email]),
    password: this.fb.control('', [Validators.required]),
  });

  protected readonly mfaCodeForm = this.fb.group({
    mfaCode: this.fb.control('', [
      Validators.required,
      Validators.pattern(/^(\d{6}|\w{5}-\w{5})$/),
      Validators.minLength(6),
      Validators.maxLength(11),
    ]),
  });

  protected readonly mfaRequired = signal(false);
  protected readonly loading = signal(false);
  protected readonly errorMessage = signal<string | undefined>(undefined);

  readonly loginConfig$ = this.auth.loginConfig();
  readonly isJWTLogin = signal(false);

  constructor() {
    this.route.queryParamMap
      .pipe(
        map((params) => params.get('email')),
        filter((email) => email !== null),
        distinctUntilChanged(),
        takeUntilDestroyed()
      )
      .subscribe((email) => this.emailPasswordForm.patchValue({email}));

    this.route.queryParamMap
      .pipe(
        map((params) => params.get('inviteSuccess')),
        filter((inviteSuccess) => inviteSuccess === 'true'),
        takeUntilDestroyed(),
        take(1)
      )
      .subscribe(() => this.toast.success('Account activated successfully. You can now log in!'));
  }

  public ngOnInit(): void {
    const reason = this.route.snapshot.queryParamMap.get('reason');
    switch (reason) {
      case 'password-reset':
        this.toast.success('Your password has been updated, you can now log in.');
        break;
      case 'session-expired':
        this.toast.success('You have been logged out because your session has expired.');
        break;
      case 'oidc-failed':
        this.toast.error('Login with this provider failed unexpectedly.');
        break;
    }

    const jwt = this.route.snapshot.queryParamMap.get('jwt');
    if (jwt) {
      this.isJWTLogin.set(true);
      this.auth.loginWithToken(jwt);
      window.location.href = '/';
    }
  }

  public async submit(): Promise<void> {
    this.emailPasswordForm.markAllAsTouched();
    this.errorMessage.set(undefined);

    const email = this.emailPasswordForm.value.email;
    const password = this.emailPasswordForm.value.password;
    const mfaCode = this.mfaCodeForm.value.mfaCode || undefined;

    if (this.emailPasswordForm.invalid || !email || !password) {
      this.emailPasswordForm.markAllAsTouched();
      return;
    }

    if (this.mfaRequired() && (this.mfaCodeForm.invalid || !mfaCode)) {
      this.mfaCodeForm.markAllAsTouched();
      return;
    }

    this.loading.set(true);

    try {
      const response = await lastValueFrom(this.auth.login(email, password, mfaCode));
      if (response.requiresMfa) {
        this.mfaRequired.set(true);
      } else if (this.auth.isCustomer()) {
        await this.router.navigate(['/home']);
      } else {
        await this.router.navigate(['/dashboard'], {queryParams: {from: 'login'}});
      }
    } catch (e) {
      this.errorMessage.set(getFormDisplayedError(e));
    } finally {
      this.loading.set(false);
    }
  }

  public reset() {
    this.emailPasswordForm.reset();
    this.mfaCodeForm.reset();
    this.mfaRequired.set(false);
    this.errorMessage.set(undefined);
  }

  protected getLoginURL(provider: string): string {
    return `/api/v1/auth/oidc/${provider}`;
  }

  protected readonly faGoogle = faGoogle;
  protected readonly faGithub = faGithub;
  protected readonly faMicrosoft = faMicrosoft;
  protected readonly faGeneric = faArrowRightToBracket;
  protected readonly faArrowRightToBracket = faArrowRightToBracket;
}
