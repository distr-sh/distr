import {Component, computed, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {CustomerOrganization, CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {OrganizationService} from '../../../services/organization.service';
import {ToastService} from '../../../services/toast.service';
import {Feature} from '../../../types/organization';
import {isProSubscription, UpgradePlan} from '../../../types/subscription';
import {PlanBadgeComponent} from '../../plan-badge.component';

interface CustomerFeatureDefinition {
  feature: CustomerOrganizationFeature;
  label: string;
  /** The feature this one builds on: enabling this one enables it as well, disabling it disables this one. */
  requires?: CustomerOrganizationFeature;
  requiresPro?: boolean;
  /** The organization feature that unlocks a Business customer feature, which decides instead of the plan. */
  unlockedBy?: Feature;
}

export const customerFeatures: CustomerFeatureDefinition[] = [
  {feature: 'deployment_targets', label: 'Deployments'},
  {feature: 'alerts', label: 'Alerts', requires: 'deployment_targets', requiresPro: true},
  {feature: 'artifacts', label: 'Artifacts'},
  {feature: 'update_notifications', label: 'Update Notifications', requiresPro: true},
  {feature: 'support_bundles', label: 'Support', requiresPro: true},
  {feature: 'vulnerabilities', label: 'Security', unlockedBy: 'vulnerabilities'},
  {feature: 'settings', label: 'Settings', unlockedBy: 'customer_settings'},
  {
    feature: 'oidc_providers',
    label: 'Identity Provider',
    requires: 'settings',
    unlockedBy: 'custom_oidc_providers',
  },
];

interface CustomerFeatureRow extends CustomerFeatureDefinition {
  enabled: boolean;
  lockedPlan?: UpgradePlan;
}

@Component({
  selector: 'app-customer-features',
  templateUrl: './customer-features.component.html',
  imports: [PlanBadgeComponent],
  host: {class: 'block'},
})
export class CustomerFeaturesComponent {
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly toast = inject(ToastService);

  public readonly customerOrganization = input.required<CustomerOrganization>();

  protected readonly canEdit = inject(AuthService).hasAnyRole('read_write', 'admin');
  protected readonly saving = signal(false);

  private readonly organization = toSignal(inject(OrganizationService).get());

  protected readonly rows = computed<CustomerFeatureRow[]>(() => {
    const granted = this.customerOrganization().features;
    const isPro = isProSubscription(this.organization()?.subscriptionType);
    const organizationFeatures = this.organization()?.features ?? [];
    const ownLock = (definition: CustomerFeatureDefinition): UpgradePlan | undefined => {
      if (definition.requiresPro && !isPro) {
        return 'pro';
      } else if (definition.unlockedBy && !organizationFeatures.includes(definition.unlockedBy)) {
        return 'business';
      } else {
        return undefined;
      }
    };
    // A feature granted before a downgrade stays switchable, so that it can still be disabled. One whose required
    // feature is locked is locked as well, since enabling it would enable the required one.
    const locks = new Map<CustomerOrganizationFeature, UpgradePlan | undefined>();
    return customerFeatures.map((definition) => {
      const enabled = granted.includes(definition.feature);
      const lockedPlan = enabled
        ? undefined
        : (ownLock(definition) ?? (definition.requires ? locks.get(definition.requires) : undefined));
      locks.set(definition.feature, lockedPlan);
      return {...definition, enabled, lockedPlan};
    });
  });

  protected async toggleFeature(row: CustomerFeatureRow, toggle: HTMLInputElement, event: Event) {
    // Every update sends the whole feature list, so a second one before the first returns would undo it.
    if (this.saving()) {
      event.preventDefault();
      return;
    }
    const customer = this.customerOrganization();
    const features = new Set(customer.features);
    if (row.enabled) {
      features.delete(row.feature);
      for (const dependent of customerFeatures.filter((it) => it.requires === row.feature)) {
        features.delete(dependent.feature);
      }
    } else {
      features.add(row.feature);
      if (row.requires) {
        features.add(row.requires);
      }
    }

    this.saving.set(true);
    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(customer.id, {
          name: customer.name,
          imageId: customer.imageId,
          features: Array.from(features),
        })
      );
      this.toast.success('Customer features updated');
    } catch (e) {
      toggle.checked = row.enabled;
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.saving.set(false);
    }
  }
}
