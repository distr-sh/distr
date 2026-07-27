import {ChangeDetectionStrategy, Component, computed, effect, inject} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faTrash} from '@fortawesome/free-solid-svg-icons';
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
  // The host must not occupy a slot in the parent grid when the section is hidden.
  host: {class: 'contents'},
})
export class CustomDomainsComponent {
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

  // Not marked required: the inputs are saved together with the organization settings form,
  // so empty domain inputs simply mean "nothing to save" and must not block that save.
  protected readonly form = this.fb.group({
    appDomain: this.fb.control('', [Validators.pattern(hostnamePattern)]),
    registryDomainEnabled: this.fb.control(false),
    registryDomain: this.fb.control({value: '', disabled: true}, [Validators.pattern(hostnamePattern)]),
  });
  protected readonly registryDomainEnabled = toSignal(this.form.controls.registryDomainEnabled.valueChanges, {
    initialValue: false,
  });

  constructor() {
    // Disabled controls are excluded from validation, so only the visible inputs block saving.
    this.form.controls.registryDomainEnabled.valueChanges.pipe(takeUntilDestroyed()).subscribe((enabled) => {
      if (enabled) {
        this.form.controls.registryDomain.enable();
      } else {
        this.form.controls.registryDomain.disable();
      }
    });
    effect(() => {
      if (this.appDomain()) {
        this.form.controls.appDomain.disable();
      } else {
        this.form.controls.appDomain.enable();
      }
    });
  }

  // Called by the organization settings save button; a no-op when no domains were entered.
  public async save() {
    const requests: {domain: string; domainType: CustomDomainType}[] = [];
    if (!this.appDomain() && this.form.controls.appDomain.enabled && this.form.controls.appDomain.value) {
      requests.push({domain: this.form.controls.appDomain.value.toLowerCase(), domainType: 'app'});
    }
    if (
      !this.registryDomain() &&
      this.form.controls.registryDomain.enabled &&
      this.form.controls.registryDomain.value
    ) {
      requests.push({domain: this.form.controls.registryDomain.value.toLowerCase(), domainType: 'registry'});
    }
    if (requests.length === 0) {
      return;
    }
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      return;
    }
    try {
      await firstValueFrom(this.customDomainsService.create(requests));
      this.form.reset();
      this.refresh$.next();
      this.toast.success('Custom domains saved. Remember to create the CNAME records at your DNS provider.');
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
