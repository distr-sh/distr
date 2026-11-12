import {Component, computed, inject, input, linkedSignal, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {BehaviorSubject, combineLatest, firstValueFrom, from, of, switchMap} from 'rxjs';
import {getRemoteEnvironment} from '../../../../env/remote';
import {getFormDisplayedError} from '../../../../util/errors';
import {ContextService} from '../../../services/context.service';
import {CustomDomainsService} from '../../../services/custom-domains.service';
import {FeatureFlagService} from '../../../services/feature-flag.service';
import {OverlayService} from '../../../services/overlay.service';
import {ToastService} from '../../../services/toast.service';
import {CustomDomain, CustomDomainType} from '../../../types/custom-domain';
import {DomainFieldComponent} from '../domain-field/domain-field.component';

@Component({
  selector: 'app-custom-domain',
  templateUrl: './custom-domain.component.html',
  host: {class: 'block space-y-4'},
  imports: [DomainFieldComponent],
})
export class CustomDomainComponent {
  private readonly customDomainsService = inject(CustomDomainsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly contextService = inject(ContextService);

  public readonly domainType = input.required<CustomDomainType>();
  public readonly customerOrganizationId = input<string>();
  public readonly label = input.required<string>();
  public readonly placeholder = input.required<string>();
  public readonly optionalCheckboxLabel = input<string>();

  private readonly remoteEnv = toSignal(from(getRemoteEnvironment()));
  protected readonly cnameTarget = computed(() => this.remoteEnv()?.customDomainTarget);

  private readonly refresh$ = new BehaviorSubject<void>(undefined);
  private readonly fetchedDomains = toSignal(
    combineLatest([
      this.featureFlags.isCustomDomainsEnabled$,
      this.featureFlags.isCustomOidcProvidersEnabled$,
      this.refresh$,
    ]).pipe(
      switchMap(([domainsEnabled, oidcEnabled]) =>
        domainsEnabled || oidcEnabled ? this.customDomainsService.list() : of([] as CustomDomain[])
      )
    ),
    {initialValue: [] as CustomDomain[]}
  );
  private readonly fetchedDomain = computed(() =>
    this.fetchedDomains().find(
      (domain) =>
        domain.domainType === this.domainType() &&
        (domain.customerOrganizationId ?? undefined) === this.customerOrganizationId()
    )
  );
  public readonly domain = linkedSignal(() => this.fetchedDomain());

  protected readonly saving = signal(false);
  protected readonly verifying = signal(false);

  protected async save(value: string) {
    this.saving.set(true);
    try {
      await firstValueFrom(
        this.customDomainsService.create(
          [{domain: value, domainType: this.domainType()}],
          this.customerOrganizationId()
        )
      );
      this.refresh$.next();
      this.contextService.reload();
      this.toast.success('Custom domain saved');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.saving.set(false);
    }
  }

  protected async remove() {
    const domain = this.domain();
    if (!domain) {
      return;
    }
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
      this.contextService.reload();
      this.toast.success('Custom domain removed');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async verify() {
    const domain = this.domain();
    if (!domain) {
      return;
    }
    this.verifying.set(true);
    try {
      const verification = await firstValueFrom(this.customDomainsService.verify(domain.id));
      this.domain.set({
        ...domain,
        verified: verification.verified,
        verifiedAt: verification.verifiedAt,
        verificationCheckedAt: verification.verificationCheckedAt,
        verificationError: verification.verificationError,
      });
      if (verification.inconclusive) {
        this.toast.error('The DNS lookup could not be completed, please try again');
      } else if (verification.verified !== domain.verified) {
        this.contextService.reload();
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.verifying.set(false);
    }
  }
}
