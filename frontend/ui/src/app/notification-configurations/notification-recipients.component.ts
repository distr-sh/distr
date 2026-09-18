import {Component, computed, inject, input, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormRecord, ReactiveFormsModule} from '@angular/forms';
import {UserAccountWithRole} from '@distr-sh/distr-sdk';
import {organizationKind} from '../../util/organization-kind';
import {AuthService} from '../services/auth.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';

interface RecipientGroup {
  label: string;
  users: UserAccountWithRole[];
}

/** The users a notification configuration sends to, grouped by the organization they belong to. */
@Component({
  selector: 'app-notification-recipients',
  template: `
    @for (group of groups(); track group.label) {
      <div class="space-y-1">
        <h4 class="text-sm dark:text-white font-semibold">{{ group.label }}</h4>
        @for (user of group.users; track user.id) {
          <label class="flex items-center w-full">
            <input type="checkbox" class="distr-checkbox" [formControl]="control().controls[user.id!]" />
            <span class="ms-2 text-sm font-medium text-gray-900 dark:text-gray-300 overflow-hidden text-ellipsis">
              {{ user.name || user.email }}
            </span>
          </label>
        }
      </div>
    }
  `,
  host: {class: 'block space-y-2'},
  imports: [ReactiveFormsModule],
})
export class NotificationRecipientsComponent {
  public readonly control = input.required<FormRecord<FormControl<boolean>>>();
  public readonly users = input.required<UserAccountWithRole[]>();

  private readonly auth = inject(AuthService);
  private readonly customerOrganizations = inject(CustomerOrganizationsService);

  private readonly customers = this.auth.isVendor()
    ? toSignal(this.customerOrganizations.getCustomerOrganizations())
    : signal(undefined).asReadonly();

  protected readonly groups = computed<RecipientGroup[]>(() => {
    const users = this.users();
    const customerNames = new Map((this.customers() ?? []).map((customer) => [customer.id, customer.name]));

    const byGroup = new Map<string, UserAccountWithRole[]>();
    for (const user of users) {
      const label =
        organizationKind(user) === 'customer'
          ? (customerNames.get(user.customerOrganizationId!) ?? 'Customer')
          : organizationKind(user) === 'partner'
            ? 'Partners'
            : 'Team';
      byGroup.set(label, [...(byGroup.get(label) ?? []), user]);
    }

    return [...byGroup.entries()].map(([label, groupUsers]) => ({label, users: groupUsers}));
  });
}
