import {DatePipe} from '@angular/common';
import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, Router} from '@angular/router';
import {CustomerOrganizationWithUsage} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faAddressBook, faGlasses, faTrash} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, of, switchMap} from 'rxjs';
import {compareBy} from '../../../../util/arrays';
import {getFormDisplayedError} from '../../../../util/errors';
import {refreshInBackground} from '../../../../util/refresh';
import {tabContext} from '../../../navigation/tabs';
import {ApplicationEntitlementsService} from '../../../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../../../services/artifact-entitlements.service';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {FeatureFlagService} from '../../../services/feature-flag.service';
import {ImageUploadService} from '../../../services/image-upload.service';
import {OverlayService} from '../../../services/overlay.service';
import {PartnerOrganizationsService} from '../../../services/partner-organizations.service';
import {ToastService} from '../../../services/toast.service';
import {AlertComponent} from '../../alert/alert.component';
import {AvatarComponent} from '../../avatar.component';
import {BadgeSelectComponent, BadgeSelectOption} from '../../badge-select/badge-select.component';
import {InlineEditComponent} from '../../inline-edit.component';
import {PageComponent} from '../../page.component';
import {RoutedTabsComponent} from '../../routed-tabs.component';
import {SpinnerComponent} from '../../spinner/spinner.component';
import {deleteCustomerConfirmConfig} from '../delete-customer-confirm';
import {customerDetailTabs} from './customer-detail-navigation';

// partner organization ids are UUIDs, so this cannot collide with one
const noPartnerOption = 'none';

@Component({
  templateUrl: './customer-detail-page.component.html',
  imports: [
    AlertComponent,
    AvatarComponent,
    BadgeSelectComponent,
    DatePipe,
    FaIconComponent,
    InlineEditComponent,
    PageComponent,
    RoutedTabsComponent,
    SpinnerComponent,
  ],
})
export class CustomerDetailPageComponent {
  protected readonly faAddressBook = faAddressBook;
  protected readonly faGlasses = faGlasses;
  protected readonly faTrash = faTrash;

  private readonly router = inject(Router);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly artifactEntitlementsService = inject(ArtifactEntitlementsService);
  private readonly applicationEntitlementsService = inject(ApplicationEntitlementsService);
  private readonly partnerOrganizationsService = inject(PartnerOrganizationsService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly auth = inject(AuthService);

  protected readonly canEdit = this.auth.hasAnyRole('read_write', 'admin');
  protected readonly canDelete = this.auth.hasAnyRole('admin') && (this.auth.isVendor() || this.auth.isPartner());

  private readonly route = inject(ActivatedRoute);
  private readonly routeParams = toSignal(this.route.params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );

  private readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.customerOrganizationId())
  );
  protected readonly loading = signal(true);
  protected readonly savingName = signal(false);
  protected readonly savingPartner = signal(false);

  /** `null` for partners and while the organization has no partner management. */
  private readonly partnerOrganizations = toSignal(
    this.auth.isVendor()
      ? this.featureFlags.isPartnerManagementEnabled$.pipe(
          switchMap((enabled) => (enabled ? this.partnerOrganizationsService.getPartnerOrganizations() : of(null)))
        )
      : of(null)
  );
  /** Undefined while there is no partner to assign the customer to. */
  protected readonly partnerOptions = computed<BadgeSelectOption[] | undefined>(() => {
    const partners = this.partnerOrganizations();
    if (!partners || (partners.length === 0 && !this.customerOrganization()?.partnerOrganizationId)) {
      return undefined;
    }
    return [
      {value: noPartnerOption, label: 'No partner', variant: 'neutral'},
      ...[...partners]
        .sort(compareBy((partner) => partner.name))
        .map<BadgeSelectOption>((partner) => ({value: partner.id, label: partner.name, variant: 'info'})),
    ];
  });
  protected readonly partnerValue = computed(
    () => this.customerOrganization()?.partnerOrganizationId ?? noPartnerOption
  );
  protected readonly partnerName = computed(() => {
    const id = this.customerOrganization()?.partnerOrganizationId;
    return id === undefined ? undefined : this.partnerOrganizations()?.find((partner) => partner.id === id)?.name;
  });

  private readonly tabContext = toSignal(tabContext());
  protected readonly tabs = computed(() => {
    const context = this.tabContext();
    return context === undefined ? [] : customerDetailTabs.resolve(context);
  });

  constructor() {
    refreshInBackground(this.customerOrganizationsService.refresh().finally(() => this.loading.set(false)));
  }

  protected async updateName(customer: CustomerOrganizationWithUsage, name: string) {
    this.savingName.set(true);
    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(customer.id, {
          name,
          imageId: customer.imageId,
          features: customer.features,
        })
      );
      this.toast.success('Customer has been updated');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.savingName.set(false);
    }
  }

  protected async updatePartner(customer: CustomerOrganizationWithUsage, value: string) {
    if (customer.partnerOrganizationId !== undefined) {
      const confirmed = await firstValueFrom(
        this.overlay.confirm({
          message: {
            message:
              value === noPartnerOption
                ? 'Are you sure you want to remove the partner from this customer?'
                : 'Are you sure you want to assign this customer to another partner?',
            alert: {
              type: 'warning',
              message: `The users of ${this.partnerName() ?? 'the current partner'} will no longer be able to manage this customer.`,
            },
          },
        })
      );
      if (!confirmed) {
        return;
      }
    }
    this.savingPartner.set(true);
    try {
      await firstValueFrom(
        this.customerOrganizationsService.assignCustomerToPartner(customer.id, {
          partnerOrganizationId: value === noPartnerOption ? undefined : value,
        })
      );
      this.toast.success('Partner has been updated');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.savingPartner.set(false);
    }
  }

  protected async uploadImage(customer: CustomerOrganizationWithUsage) {
    const imageId = await firstValueFrom(this.imageUploadService.showDialog({imageUrl: customer.imageUrl}));
    // The update replaces the whole customer, so it has to carry what changed while the dialog was open.
    const current = this.customerOrganization();
    if (!imageId || current?.id !== customer.id || imageId === current.imageId) {
      return;
    }
    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(current.id, {
          name: current.name,
          imageId,
          features: current.features,
        })
      );
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async openCustomerPortal(customer: CustomerOrganizationWithUsage) {
    try {
      await this.auth.enterLens(customer.id);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async delete(customer: CustomerOrganizationWithUsage) {
    if (!(await firstValueFrom(this.overlay.confirm(deleteCustomerConfirmConfig(customer))))) {
      return;
    }
    try {
      await firstValueFrom(this.customerOrganizationsService.deleteCustomerOrganization(customer.id));
      this.artifactEntitlementsService.refresh();
      this.applicationEntitlementsService.refresh();
      await this.router.navigate(['/', 'customers']);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
