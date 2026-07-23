import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus, faTrash} from '@fortawesome/free-solid-svg-icons';
import {BehaviorSubject, combineLatest, firstValueFrom, of, switchMap} from 'rxjs';
import {fromPromise} from 'rxjs/internal/observable/innerFrom';
import {getRemoteEnvironment} from '../../env/remote';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {CustomDomainsService} from '../services/custom-domains.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {CustomDomain, CustomDomainType} from '../types/custom-domain';

// RFC-1123 hostname: dot-separated labels of lowercase alphanumerics and hyphens
// (not at the start or end of a label), at least two labels.
const hostnamePattern = /^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$/;

@Component({
  selector: 'app-custom-domains',
  templateUrl: './custom-domains.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective],
})
export class CustomDomainsComponent {
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;

  private readonly customDomainsService = inject(CustomDomainsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;

  private readonly remoteEnv = toSignal(fromPromise(getRemoteEnvironment()));
  protected readonly appCnameTarget = computed(() => this.remoteEnv()?.customDomainAppCnameTarget);
  protected readonly registryCnameTarget = computed(
    () => this.remoteEnv()?.customDomainRegistryCnameTarget ?? this.appCnameTarget()
  );

  protected readonly visible = computed(
    () =>
      this.featureFlags.isCustomDomainsEnabled() &&
      this.auth.isVendor() &&
      this.auth.hasRole('admin') &&
      !!this.appCnameTarget()
  );

  private readonly refresh$ = new BehaviorSubject<void>(undefined);
  protected readonly domains = toSignal(
    combineLatest([this.featureFlags.isCustomDomainsEnabled$, this.refresh$]).pipe(
      switchMap(([enabled]) => (enabled ? this.customDomainsService.list() : of([] as CustomDomain[])))
    ),
    {initialValue: [] as CustomDomain[]}
  );
  protected readonly appDomain = computed(() => this.domains().find((d) => d.domainType === 'app'));
  protected readonly registryDomain = computed(() => this.domains().find((d) => d.domainType === 'registry'));

  protected readonly appDomainForm = this.fb.group({
    domain: this.fb.control('', [Validators.required, Validators.pattern(hostnamePattern)]),
  });
  protected readonly registryDomainForm = this.fb.group({
    domain: this.fb.control('', [Validators.required, Validators.pattern(hostnamePattern)]),
  });

  protected async add(domainType: CustomDomainType) {
    const form = domainType === 'app' ? this.appDomainForm : this.registryDomainForm;
    form.markAllAsTouched();
    if (form.invalid) {
      return;
    }
    try {
      await firstValueFrom(
        this.customDomainsService.create({domain: form.controls.domain.value.toLowerCase(), domainType})
      );
      form.reset();
      this.refresh$.next();
      this.toast.success('Custom domain added. Remember to create the CNAME record at your DNS provider.');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async remove(domain: CustomDomain) {
    const confirmed = await firstValueFrom(
      this.overlay.confirm(
        `Really remove ${domain.domain}? The domain will stop working and its TLS certificate will no longer be renewed.`
      )
    );
    if (!confirmed) {
      return;
    }
    try {
      await firstValueFrom(this.customDomainsService.delete(domain.id));
      this.refresh$.next();
      this.toast.success('Custom domain removed');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
