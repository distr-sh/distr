import {ChangeDetectionStrategy, Component, computed, inject, signal, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk} from '@fortawesome/free-solid-svg-icons';
import {AlertComponent} from '../components/alert/alert.component';
import {BrandingFormComponent} from '../components/branding/branding-form.component';
import {PageComponent} from '../components/page.component';
import {canOpenSettingsTab} from '../organization-settings/organization-settings-navigation';
import {AuthService} from '../services/auth.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OrganizationService} from '../services/organization.service';
import {canOpenSubscription} from '../subscription/subscription.guard';

@Component({
  selector: 'app-organization-branding',
  templateUrl: './organization-branding.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [BrandingFormComponent, FaIconComponent, RouterLink, PageComponent, AlertComponent],
})
export class OrganizationBrandingComponent {
  protected readonly faFloppyDisk = faFloppyDisk;

  protected readonly auth = inject(AuthService);
  private readonly organizationService = inject(OrganizationService);
  private readonly featureFlags = inject(FeatureFlagService);

  private readonly organization = toSignal(this.organizationService.get());
  protected readonly canOpenSubscription = toSignal(canOpenSubscription(), {initialValue: false});
  protected readonly canOpenCustomDomains = toSignal(canOpenSettingsTab('identity-provider'), {initialValue: false});

  protected readonly customDomainsSelfService = this.featureFlags.isCustomDomainsEnabled;
  // Business plan upsell, shown only on the plans that can actually upgrade to it
  protected readonly showCustomDomainsUpsell = computed(() => {
    const subscriptionType = this.organization()?.subscriptionType;
    return !this.customDomainsSelfService() && (subscriptionType === 'pro' || subscriptionType === 'trial');
  });

  private readonly brandingForm = viewChild.required(BrandingFormComponent);

  protected readonly formLoading = signal(false);

  protected async save() {
    this.formLoading.set(true);
    try {
      await this.brandingForm().save();
    } finally {
      this.formLoading.set(false);
    }
  }
}
