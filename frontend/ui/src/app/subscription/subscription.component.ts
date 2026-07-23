import {GlobalPositionStrategy, OverlayModule} from '@angular/cdk/overlay';
import {CommonModule} from '@angular/common';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {NonNullableFormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faCreditCard, faShoppingCart} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {WEBSITE_URL} from '../../constants';
import {getFormDisplayedError} from '../../util/errors';
import {never} from '../../util/exhaust';
import {DeleteOrganizationComponent} from '../components/delete-organization/delete-organization.component';
import {AuthService} from '../services/auth.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OrganizationService} from '../services/organization.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {SubscriptionService} from '../services/subscription.service';
import {ToastService} from '../services/toast.service';
import {SubscriptionInfo, SubscriptionPeriod, SubscriptionType, UNLIMITED_QTY} from '../types/subscription';
import {PendingSubscriptionUpdate, SubscriptionUpdateModalComponent} from './subscription-update-modal.component';

@Component({
  selector: 'app-subscription',
  templateUrl: './subscription.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    FaIconComponent,
    ReactiveFormsModule,
    CommonModule,
    OverlayModule,
    SubscriptionUpdateModalComponent,
    DeleteOrganizationComponent,
  ],
})
export class SubscriptionComponent implements OnInit {
  protected readonly faCheck = faCheck;
  protected readonly faCreditCard = faCreditCard;
  protected readonly faShoppingCart = faShoppingCart;

  protected readonly unlimited = UNLIMITED_QTY;
  protected readonly websiteUrl = WEBSITE_URL;

  // Feature lists mirroring the pricing cards on https://distr.sh/pricing/
  protected readonly proPlanFeatures = [
    'Docker + Kubernetes deployment agents',
    'Customer Portal with installation instructions',
    'RBAC + Login with Google, GitHub & Microsoft',
    'License Management',
    'Deployment Alerts',
    '1TB container registry with FGAC',
    'Custom Branding for your Customer Portal',
    '7-day log retention',
    'Free Onboarding Call + Private Slack',
  ];
  protected readonly businessPlanFeatures = [
    'Everything in Pro',
    'Reseller / Partner Organizations',
    'License Templates',
    'Custom Domains (Full White Label)',
    'Enterprise SSO (bring-your-own OIDC)',
    '5TB container registry with FGAC',
    '30-day log retention',
    'Priority support',
    'White Glove Onboarding',
  ];

  protected readonly auth = inject(AuthService);
  private readonly subscriptionService = inject(SubscriptionService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(NonNullableFormBuilder);

  protected readonly isSubscriptionExpired = inject(OrganizationService).isSubscriptionExpired;
  protected readonly isPartnerManagementEnabled = inject(FeatureFlagService).isPartnerManagementEnabled;

  protected subscriptionInfo = signal<SubscriptionInfo | undefined>(undefined);
  protected pendingUpdate = signal<PendingSubscriptionUpdate | undefined>(undefined);

  private modal?: DialogRef;

  protected readonly updateModal = viewChild.required<TemplateRef<unknown>>('updateModal');

  protected readonly form = this.fb.group({
    subscriptionType: this.fb.control<SubscriptionType>('pro', [Validators.required]),
    subscriptionPeriod: this.fb.control<SubscriptionPeriod>('monthly', [Validators.required]),
    userAccountQuantity: this.fb.control<number>(1, [Validators.required, Validators.min(1)]),
    customerOrganizationQuantity: this.fb.control<number>(1, [Validators.required, Validators.min(1)]),
  });

  protected readonly formValues = toSignal(this.form.valueChanges, {initialValue: this.form.value});

  protected readonly hasQuantitiesChanged = computed(() => {
    const info = this.subscriptionInfo();
    const values = this.formValues();

    if (!info) {
      return false;
    }

    return (
      values.userAccountQuantity !== info.subscriptionUserAccountQuantity ||
      values.customerOrganizationQuantity !== info.subscriptionCustomerOrganizationQuantity
    );
  });

  async ngOnInit() {
    try {
      const info = await firstValueFrom(this.subscriptionService.get());
      this.subscriptionInfo.set(info);

      // Pre-fill form with current subscription values or defaults
      const defaultType = info.subscriptionType === 'trial' ? 'pro' : info.subscriptionType;

      this.form.patchValue({
        subscriptionType: defaultType,
        userAccountQuantity:
          info.subscriptionUserAccountQuantity !== UNLIMITED_QTY
            ? info.subscriptionUserAccountQuantity
            : info.currentUserAccountCount,
        customerOrganizationQuantity:
          info.subscriptionCustomerOrganizationQuantity !== UNLIMITED_QTY
            ? info.subscriptionCustomerOrganizationQuantity
            : info.currentCustomerOrganizationCount,
      });
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  getPreviewPrice(): number {
    const values = this.form.getRawValue();
    return this.getPreviewPriceFor(values.subscriptionType, values.subscriptionPeriod);
  }

  getPreviewPriceFor(subscriptionType: SubscriptionType, subscriptionPeriod: SubscriptionPeriod): number {
    const values = this.form.getRawValue();
    const userQty = values.userAccountQuantity;
    const customerQty = values.customerOrganizationQuantity;

    return this.calculatePrice(subscriptionType, subscriptionPeriod, userQty, customerQty);
  }

  calculatePrice(
    subscriptionType: SubscriptionType,
    subscriptionPeriod: SubscriptionPeriod,
    userQty: number,
    customerQty: number
  ) {
    let userPrice = 0;
    let customerPrice = 0;

    if (subscriptionType === 'pro') {
      userPrice = subscriptionPeriod === 'monthly' ? 29 : 288;
      customerPrice = subscriptionPeriod === 'monthly' ? 69 : 672;
    } else if (subscriptionType === 'business') {
      userPrice = subscriptionPeriod === 'monthly' ? 39 : 384;
      customerPrice = subscriptionPeriod === 'monthly' ? 159 : 1536;
    }
    return userPrice * userQty + customerPrice * customerQty;
  }

  async checkout() {
    this.form.markAllAsTouched();
    if (this.form.valid) {
      try {
        const values = this.form.getRawValue();
        const request = {
          subscriptionType: values.subscriptionType,
          subscriptionPeriod: values.subscriptionPeriod,
          subscriptionUserAccountQuantity: values.userAccountQuantity,
          subscriptionCustomerOrganizationQuantity: values.customerOrganizationQuantity,
        };

        // Call the checkout endpoint which will redirect to Stripe
        await this.subscriptionService.checkout(request);
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      }
    }
  }

  async updateQuantities() {
    this.form.markAllAsTouched();
    if (this.form.valid) {
      const info = this.subscriptionInfo();
      if (!info) {
        return;
      }

      // Calculate current and new prices
      const oldPrice = this.calculatePriceFor(info);
      const newPrice = this.getPreviewPriceFor(info.subscriptionType, info.subscriptionPeriod);

      // Set pending update and show confirmation modal
      const values = this.form.getRawValue();
      this.pendingUpdate.set({
        userAccountQuantity: values.userAccountQuantity,
        customerOrganizationQuantity: values.customerOrganizationQuantity,
        newPrice,
        oldPrice,
        subscriptionPeriod: info.subscriptionPeriod,
      });

      this.hideModal();
      this.modal = this.overlay.showModal(this.updateModal(), {
        hasBackdrop: true,
        backdropStyleOnly: true,
        positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
      });
    }
  }

  onModalConfirmed(updatedInfo: SubscriptionInfo) {
    this.subscriptionInfo.set(updatedInfo);
    this.hideModal();
  }

  hideModal() {
    this.modal?.close();
  }

  private calculatePriceFor(info: SubscriptionInfo): number {
    if (info.subscriptionUserAccountQuantity == null || info.subscriptionCustomerOrganizationQuantity == null) {
      return 0;
    }

    return this.calculatePrice(
      info.subscriptionType,
      info.subscriptionPeriod,
      info.subscriptionUserAccountQuantity,
      info.subscriptionCustomerOrganizationQuantity
    );
  }

  getPlanLimits(plan: SubscriptionType): {customers: string; users: string; deployments: string} {
    const limits = this.getPlanLimitsObject(plan);
    if (!limits) {
      return {customers: '', users: '', deployments: ''};
    }

    return {
      customers:
        limits.maxCustomerOrganizations === UNLIMITED_QTY
          ? 'Unlimited customers'
          : `Up to ${limits.maxCustomerOrganizations} customer${limits.maxCustomerOrganizations > 1 ? 's' : ''}`,
      users:
        limits.maxUsersPerCustomerOrganization === UNLIMITED_QTY
          ? 'Unlimited users per customer'
          : `Up to ${limits.maxUsersPerCustomerOrganization} user account${limits.maxUsersPerCustomerOrganization > 1 ? 's' : ''} per customer`,
      deployments:
        limits.maxDeploymentsPerCustomerOrganization === UNLIMITED_QTY
          ? 'Unlimited deployments per customer'
          : `${limits.maxDeploymentsPerCustomerOrganization} active deployment${limits.maxDeploymentsPerCustomerOrganization > 1 ? 's' : ''} per customer`,
    };
  }

  private getPlanLimitsObject(subscriptionType: SubscriptionType) {
    const info = this.subscriptionInfo();
    if (!info) {
      return undefined;
    }
    return info.limits[subscriptionType];
  }

  getPlanLimit(
    subscriptionType: SubscriptionType,
    metric: 'customerOrganizations' | 'usersPerCustomer' | 'deploymentsPerCustomer'
  ): string | number {
    const limits = this.getPlanLimitsObject(subscriptionType);
    if (!limits) {
      return '';
    }

    switch (metric) {
      case 'customerOrganizations':
        return limits.maxCustomerOrganizations === UNLIMITED_QTY ? 'unlimited' : limits.maxCustomerOrganizations;
      case 'usersPerCustomer':
        return limits.maxUsersPerCustomerOrganization === UNLIMITED_QTY
          ? 'unlimited'
          : limits.maxUsersPerCustomerOrganization;
      case 'deploymentsPerCustomer':
        return limits.maxDeploymentsPerCustomerOrganization === UNLIMITED_QTY
          ? 'unlimited'
          : limits.maxDeploymentsPerCustomerOrganization;
      default:
        return never(metric);
    }
  }

  getCurrentPlanLimit(
    metric: 'customerOrganizations' | 'usersPerCustomer' | 'deploymentsPerCustomer'
  ): string | number {
    const info = this.subscriptionInfo();
    if (!info) {
      return '';
    }
    return this.getPlanLimit(info.subscriptionType, metric);
  }

  getPlanDisplayName(subscriptionType: SubscriptionType): string {
    switch (subscriptionType) {
      case 'community':
        return 'Distr Community Edition';
      case 'trial':
        return 'Distr Pro Unlimited Trial';
      case 'pro':
        return 'Distr Pro';
      case 'business':
        return 'Distr Business';
      case 'enterprise':
        return 'Distr Enterprise';
      default:
        return never(subscriptionType);
    }
  }

  isTrialSubscription(): boolean {
    const info = this.subscriptionInfo();
    return info?.subscriptionType === 'trial';
  }

  hasActiveSubscription(): boolean {
    const info = this.subscriptionInfo();
    return info?.subscriptionType === 'pro' || info?.subscriptionType === 'business';
  }

  canUpgradeToBusiness(): boolean {
    return this.subscriptionInfo()?.subscriptionType === 'pro';
  }

  async upgradeToBusiness() {
    this.form.markAllAsTouched();
    if (this.form.valid) {
      const info = this.subscriptionInfo();
      if (!info) {
        return;
      }

      const values = this.form.getRawValue();
      this.pendingUpdate.set({
        subscriptionType: 'business',
        userAccountQuantity: values.userAccountQuantity,
        customerOrganizationQuantity: values.customerOrganizationQuantity,
        oldPrice: this.calculatePriceFor(info),
        newPrice: this.calculatePrice(
          'business',
          info.subscriptionPeriod,
          values.userAccountQuantity,
          values.customerOrganizationQuantity
        ),
        subscriptionPeriod: info.subscriptionPeriod,
      });

      this.hideModal();
      this.modal = this.overlay.showModal(this.updateModal(), {
        hasBackdrop: true,
        backdropStyleOnly: true,
        positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
      });
    }
  }

  async manageSubscription() {
    try {
      await this.subscriptionService.openBillingPortal();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
