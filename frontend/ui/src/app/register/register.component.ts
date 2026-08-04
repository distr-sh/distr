import {ChangeDetectionStrategy, Component, computed, inject, OnInit, signal, viewChild} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {OidcButtonsComponent} from '../components/oidc-buttons.component';
import {PortalLogoComponent} from '../components/portal-logo/portal-logo.component';
import {TurnstileComponent} from '../components/turnstile.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {PlaceholderDirective} from '../directives/placeholder.directive';
import {AuthService} from '../services/auth.service';
import {PortalService} from '../services/portal.service';

@Component({
  selector: 'app-register',
  imports: [
    RouterLink,
    ReactiveFormsModule,
    AutotrimDirective,
    PlaceholderDirective,
    OidcButtonsComponent,
    PortalLogoComponent,
    TurnstileComponent,
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './register.component.html',
})
export class RegisterComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly auth = inject(AuthService);
  private readonly portal = inject(PortalService);
  private readonly loginConfig = this.portal.loginConfig;

  protected readonly errorMessage = signal<string | undefined>(undefined);
  protected readonly loading = signal(false);

  protected readonly turnstileSiteKey = computed(() => this.loginConfig().turnstileSiteKey);
  private readonly turnstile = viewChild(TurnstileComponent);
  private readonly turnstileToken = signal<string | undefined>(undefined);

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

  protected onTurnstileToken(token: string | undefined): void {
    this.turnstileToken.set(token);
  }

  public async submit(): Promise<void> {
    this.form.markAllAsTouched();
    this.errorMessage.set(undefined);
    if (!this.form.valid) {
      return;
    }
    // Wait for the portal config to settle: before it loads (and whenever its fetch fails) the site key is absent,
    // which is indistinguishable from Turnstile being disabled, so a submit racing ahead of it would skip a challenge
    // the backend still requires and fail with a captcha error that no widget can recover from.
    const {turnstileSiteKey} = (await firstValueFrom(this.portal.portal$)).loginConfig;
    const turnstileToken = this.turnstileToken();
    if (turnstileSiteKey && !turnstileToken) {
      this.errorMessage.set('Please complete the challenge to prove that you are human');
      return;
    }

    this.loading.set(true);
    const value = this.form.value;
    try {
      await firstValueFrom(
        this.auth.register(value.email!, value.name, value.organizationName, value.password!, turnstileToken)
      );
      location.assign('/');
    } catch (e) {
      this.errorMessage.set(getFormDisplayedError(e));
      this.loading.set(false);
      // A token can only be redeemed once, so a retry needs a new challenge.
      this.turnstile()?.reset();
    }
  }
}
