import {ChangeDetectionStrategy, Component, DestroyRef, inject, OnInit} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {firstValueFrom, take} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {OidcButtonsComponent} from '../components/oidc-buttons.component';
import {PortalLogoComponent} from '../components/portal-logo/portal-logo.component';
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
  ],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './register.component.html',
})
export class RegisterComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  private readonly portalService = inject(PortalService);
  private readonly destroyRef = inject(DestroyRef);

  errorMessage?: string;
  loading = false;
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

    // The login page only hides the sign-up link, so this page would otherwise render and submit a request that the
    // server refuses - on an instance with registration disabled as much as on a vendor's custom domain.
    this.portalService.portal$.pipe(take(1), takeUntilDestroyed(this.destroyRef)).subscribe(({loginConfig}) => {
      if (!loginConfig.registrationEnabled) {
        this.router.navigate(['/login'], {replaceUrl: true});
      }
    });
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
