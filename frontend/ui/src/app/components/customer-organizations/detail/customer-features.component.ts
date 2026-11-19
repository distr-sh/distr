import {Component, computed, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {CustomerOrganization, CustomerOrganizationFeature} from '@distr-sh/distr-sdk';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../../util/errors';
import {tabContext} from '../../../navigation/tabs';
import {AuthService} from '../../../services/auth.service';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {ToastService} from '../../../services/toast.service';
import {
  CustomerFeatureDefinition,
  customerFeatures,
  customerFeaturesLock,
  switchCustomerFeature,
  unmetCustomerFeatureRequirement,
} from '../../../types/customer-features';
import {isLocked, upgradePlan} from '../../../types/feature-lock';
import {UpgradePlan} from '../../../types/subscription';
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

  private readonly context = toSignal(tabContext());

  protected readonly rows = computed<CustomerFeatureRow[]>(() => {
    const granted = this.customerOrganization().features;
    const context = this.context();
    // A feature granted before a downgrade stays switchable, so that it can still be disabled. One whose required
    // feature is locked is locked as well, since enabling it would enable the required one.
    const locks = new Map<CustomerOrganizationFeature, UpgradePlan | undefined>();
    return customerFeatures.map((definition) => {
      const enabled = granted.includes(definition.feature);
      const {requires} = definition;
      const locked = customerFeaturesLock(definition.feature);
      const ownLock = locked && (!context || isLocked(locked, context.features)) ? upgradePlan(locked) : undefined;
      const lockedPlan = enabled ? undefined : (ownLock ?? (requires ? locks.get(requires) : undefined));
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
