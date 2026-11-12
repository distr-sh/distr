import {Component, computed, inject, input} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlug, faPlus, faTrash} from '@fortawesome/free-solid-svg-icons';
import {BehaviorSubject, combineLatest, firstValueFrom, map, of, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {ClickableRowDirective} from '../../../directives/clickable-row.directive';
import {AuthService} from '../../../services/auth.service';
import {CustomOidcService} from '../../../services/custom-oidc.service';
import {FeatureFlagService} from '../../../services/feature-flag.service';
import {OrganizationService} from '../../../services/organization.service';
import {OverlayService} from '../../../services/overlay.service';
import {ToastService} from '../../../services/toast.service';
import {CustomDomain} from '../../../types/custom-domain';
import {CustomOidcConfiguration} from '../../../types/custom-oidc';
import {ClipComponent} from '../../clip.component';
import {
  IdentityProviderModalComponent,
  IdentityProviderModalData,
} from '../identity-provider-modal/identity-provider-modal.component';

@Component({
  selector: 'app-identity-providers',
  templateUrl: './identity-providers.component.html',
  host: {class: 'block space-y-4'},
  imports: [FaIconComponent, ClickableRowDirective, ClipComponent, RouterLink],
})
export class IdentityProvidersComponent {
  protected readonly faPlug = faPlug;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;

  private readonly customOidcService = inject(CustomOidcService);
  private readonly organizationService = inject(OrganizationService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);

  public readonly domain = input<CustomDomain>();
  public readonly customerOrganizationId = input<string>();

  protected readonly enabled = computed(() => this.featureFlags.isCustomOidcProvidersEnabled());
  protected readonly viewerIsVendor = computed(() => this.auth.isVendor());

  private readonly organizationSlug = toSignal(
    this.organizationService.get().pipe(map((organization) => organization.slug)),
    {initialValue: undefined}
  );
  protected readonly configurable = computed(() => !!this.organizationSlug());

  private readonly domainId = computed(() => this.domain()?.id);
  private readonly refresh$ = new BehaviorSubject<void>(undefined);
  protected readonly configurations = toSignal(
    combineLatest([this.featureFlags.isCustomOidcProvidersEnabled$, toObservable(this.domainId), this.refresh$]).pipe(
      switchMap(([enabled, domainId]) =>
        enabled && domainId
          ? this.customOidcService
              .list()
              .pipe(
                map((response) =>
                  response.configurations.filter((configuration) => configuration.customDomainId === domainId)
                )
              )
          : of([] as CustomOidcConfiguration[])
      )
    ),
    {initialValue: [] as CustomOidcConfiguration[]}
  );

  private readonly togglingIds = new Set<string>();

  protected async showDialog(configuration?: CustomOidcConfiguration) {
    const domain = this.domain();
    if (!domain) {
      return;
    }
    const data: IdentityProviderModalData = {
      domain,
      customerOrganizationId: this.customerOrganizationId(),
      configuration,
    };
    const saved = await firstValueFrom(
      this.overlay.showModal<boolean>(IdentityProviderModalComponent, {data}).result()
    );
    if (saved) {
      this.refresh$.next();
    }
  }

  protected async toggleEnabled(configuration: CustomOidcConfiguration, toggle: HTMLInputElement, event: Event) {
    if (this.togglingIds.has(configuration.id)) {
      event.preventDefault();
      return;
    }
    this.togglingIds.add(configuration.id);
    try {
      await firstValueFrom(
        this.customOidcService.update(configuration.id, {
          customDomainId: configuration.customDomainId,
          name: configuration.name,
          slug: configuration.slug,
          enabled: !configuration.enabled,
          issuer: configuration.issuer,
          clientId: configuration.clientId,
          scopes: configuration.scopes,
          pkceEnabled: configuration.pkceEnabled,
          spInitiated: configuration.spInitiated,
          createUnknownUsers: configuration.createUnknownUsers,
          defaultUserRole: configuration.defaultUserRole,
          allowedEmailDomains: configuration.allowedEmailDomains,
        })
      );
      this.refresh$.next();
    } catch (e) {
      toggle.checked = configuration.enabled;
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.togglingIds.delete(configuration.id);
    }
  }

  protected async test(configuration: CustomOidcConfiguration) {
    try {
      await firstValueFrom(this.customOidcService.test(configuration.id));
      this.toast.success(`${configuration.name} serves a valid OpenID configuration`);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async remove(configuration: CustomOidcConfiguration) {
    const confirmed = await firstValueFrom(
      this.overlay.confirm(
        `Really delete ${configuration.name}? Everyone who signs in through it loses that login, and users without ` +
          `a password will have to reset it.`
      )
    );
    if (!confirmed) {
      return;
    }
    try {
      await firstValueFrom(this.customOidcService.delete(configuration.id));
      this.refresh$.next();
      this.toast.success('Identity provider deleted');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
