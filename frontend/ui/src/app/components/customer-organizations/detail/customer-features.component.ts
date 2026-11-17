import {Component, computed, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {CustomerOrganization, CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {OrganizationService} from '../../../services/organization.service';
import {ToastService} from '../../../services/toast.service';
import {
  CustomerFeatureDefinition,
  customerFeatures,
  switchCustomerFeature,
  unmetCustomerFeatureRequirement,
} from '../../../types/customer-features';
import {isProSubscription, UpgradePlan} from '../../../types/subscription';
import {PlanBadgeComponent} from '../../plan-badge.component';

interface CustomerFeatureRow extends CustomerFeatureDefinition {
  enabled: boolean;
  lockedPlan?: UpgradePlan;
  unmetRequirement?: string;
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
      const unmetRequirement = enabled
        ? undefined
        : unmetCustomerFeatureRequirement(granted, definition.feature, {enablesRequired: true});
      return {...definition, enabled, lockedPlan, unmetRequirement};
    });
  });

  protected async toggleFeature(row: CustomerFeatureRow, toggle: HTMLInputElement, event: Event) {
    // Every update sends the whole feature list, so a second one before the first returns would undo it.
    if (this.saving()) {
      event.preventDefault();
      return;
    }
    const customer = this.customerOrganization();
    this.saving.set(true);
    try {
      await firstValueFrom(
        this.customerOrganizationsService.updateCustomerOrganization(customer.id, {
          name: customer.name,
          imageId: customer.imageId,
          features: switchCustomerFeature(customer.features, row.feature, !row.enabled),
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
