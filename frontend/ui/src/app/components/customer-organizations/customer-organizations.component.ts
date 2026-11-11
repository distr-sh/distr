import {AsyncPipe, DecimalPipe} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router, RouterLink} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faAddressBook, faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, map, startWith} from 'rxjs';
import {refreshInBackground} from '../../../util/refresh';
import {AuthService} from '../../services/auth.service';
import {CustomerOrganizationsService} from '../../services/customer-organizations.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {AlertComponent} from '../alert/alert.component';
import {AvatarComponent} from '../avatar.component';
import {PageComponent} from '../page.component';
import {QuotaLimitComponent} from '../quota-limit.component';
import {SearchBarComponent} from '../search-bar.component';

@Component({
  templateUrl: './customer-organizations.component.html',
  imports: [
    ReactiveFormsModule,
    FontAwesomeModule,
    AsyncPipe,
    DecimalPipe,
    RouterLink,
    QuotaLimitComponent,
    AvatarComponent,
    PageComponent,
    SearchBarComponent,
    AlertComponent,
  ],
})
export class CustomerOrganizationsComponent {
  protected readonly faPlus = faPlus;
  protected readonly faAddressBook = faAddressBook;
  protected readonly faXmark = faXmark;

  private readonly router = inject(Router);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly organizationService = inject(OrganizationService);
  protected readonly featureFlags = inject(FeatureFlagService);
  protected readonly auth = inject(AuthService);

  private readonly organization = toSignal(this.organizationService.get());
  protected readonly limit = computed(() => this.organization()?.subscriptionCustomerOrganizationQuantity);
  protected readonly currentCustomerOrganizationCount = computed(
    () => this.organization()?.currentCustomerOrganizationCount
  );

  protected readonly filterForm = this.fb.group({
    search: this.fb.control(''),
  });
  protected readonly customerOrganizations = toSignal(
    combineLatest([
      this.filterForm.valueChanges.pipe(
        map((filter) => filter.search ?? ''),
        startWith('')
      ),
      this.customerOrganizationsService.getCustomerOrganizations(),
    ]).pipe(
      map(([filter, organizations]) =>
        filter.length > 0
          ? organizations.filter((organization) => organization.name.toLowerCase().includes(filter.toLowerCase()))
          : organizations
      )
    )
  );

  private readonly createCustomerDialog = viewChild.required<TemplateRef<unknown>>('createCustomerDialog');
  private modalRef?: DialogRef;
  protected readonly createForm = this.fb.group({
    name: this.fb.control('', [Validators.required]),
  });
  protected readonly createFormLoading = signal(false);

  constructor() {
    refreshInBackground(this.customerOrganizationsService.refresh());
  }

  protected showCreateDialog() {
    this.closeCreateDialog();
    this.modalRef = this.overlay.showModal(this.createCustomerDialog());
  }

  protected closeCreateDialog(reset: boolean = true): void {
    this.modalRef?.close();

    if (reset) {
      this.createForm.reset();
    }
  }

  protected async submitCreateForm() {
    this.createForm.markAllAsTouched();

    if (this.createForm.invalid) {
      return;
    }

    this.createFormLoading.set(true);

    try {
      const customer = await firstValueFrom(
        this.customerOrganizationsService.createCustomerOrganization({
          name: this.createForm.value.name!,
        })
      );

      this.closeCreateDialog();
      await this.router.navigate(['/', 'customers', customer.id]);
    } finally {
      this.createFormLoading.set(false);
    }
  }
}
