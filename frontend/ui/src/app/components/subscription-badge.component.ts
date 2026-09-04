import {Component, computed, input} from '@angular/core';
import dayjs from 'dayjs';
import {isExpiredSubscription, SubscriptionType} from '../types/subscription';

@Component({
  selector: 'app-subscription-badge',
  styleUrl: './subscription-badge.component.scss',
  host: {
    '[class]': 'planClass()',
    '[attr.title]': 'endDateHint()',
  },
  template: `{{ plan() }}`,
})
export class SubscriptionBadgeComponent {
  public readonly plan = input.required<SubscriptionType>();
  public readonly endsAt = input<string>();

  private readonly expired = computed(() => isExpiredSubscription(this.plan(), this.endsAt()));

  // an expired subscription is red whatever plan it was on, and community and a running trial take
  // the gray of the stylesheet's base rule, so neither of them needs a class
  protected readonly planClass = computed(() => (this.expired() ? 'plan-expired' : `plan-${this.plan()}`));

  protected readonly endDateHint = computed(() => {
    const plan = this.plan();
    const endsAt = this.endsAt();
    if (plan === 'community' || !endsAt) {
      return null;
    }
    const label = plan.charAt(0).toUpperCase() + plan.slice(1);
    // a trial runs out where a paid plan renews, and the subscription page words it the same way
    const verb = this.expired() ? 'expired' : plan === 'trial' ? 'expires' : 'renews';
    return `${label} ${verb} on ${dayjs(endsAt).format('MMMM D, YYYY')}`;
  });
}
