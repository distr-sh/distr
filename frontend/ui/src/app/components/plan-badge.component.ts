import {Component, input} from '@angular/core';
import {SubscriptionType} from '../types/subscription';

@Component({
  selector: 'app-plan-badge',
  template: `<span class="distr-plan-badge capitalize">{{ plan() }}</span>`,
})
export class PlanBadgeComponent {
  public readonly plan = input.required<SubscriptionType>();
}
