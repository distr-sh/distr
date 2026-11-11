import {DatePipe} from '@angular/common';
import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {CustomerOrganizationWithUsage} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faAddressBook, faGlasses, faTrash} from '@fortawesome/free-solid-svg-icons';
import {filter, firstValueFrom, map} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {refreshInBackground} from '../../../util/refresh';
import {ApplicationEntitlementsService} from '../../services/application-entitlements.service';
import {ArtifactEntitlementsService} from '../../services/artifact-entitlements.service';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {ImageUploadService} from '../../services/image-upload.service';
import {OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {AlertComponent} from '../alert/alert.component';
import {AvatarComponent} from '../avatar.component';
import {CustomerBreadcrumbComponent} from '../customer-breadcrumb.component';
import {PageComponent} from '../page.component';
import {SpinnerComponent} from '../spinner/spinner.component';
import {TabBarComponent, TabItem} from '../tab-bar.component';
import {CustomerDetailTab, customerDetailTabs} from './customer-detail-tabs';
import {deleteCustomerConfirmConfig} from './delete-customer-confirm';

@Component({
  templateUrl: './customer-detail-page.component.html',
  imports: [
    AlertComponent,
    AvatarComponent,
    CustomerBreadcrumbComponent,
    DatePipe,
    FaIconComponent,
    PageComponent,
    RouterOutlet,
    SpinnerComponent,
    TabBarComponent,
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

  protected readonly tabs = customerDetailTabs;
  protected readonly activeTab = toSignal(
    this.router.events.pipe(
      filter((event) => event instanceof NavigationEnd),
      map(() => this.tabFromRoute())
    ),
    {initialValue: this.tabFromRoute()}
  );

  constructor() {
    refreshInBackground(this.customerOrganizationsService.refresh().finally(() => this.loading.set(false)));
  }

  protected async uploadImage(customer: CustomerOrganizationWithUsage) {
    const imageId = await firstValueFrom(this.imageUploadService.showDialog({imageUrl: customer.imageUrl}));
    if (!imageId || imageId === customer.imageId) {
      return;
    }
    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(customer.id, {
          name: customer.name,
          imageId,
          features: customer.features,
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

  protected onTabClick(tab: TabItem<CustomerDetailTab>) {
    this.router.navigate([tab.id], {relativeTo: this.route});
  }

  private tabFromRoute(): CustomerDetailTab {
    const path = this.route.snapshot.firstChild?.routeConfig?.path;
    return this.tabs.find((tab) => tab.id === path)?.id ?? 'overview';
  }
}
