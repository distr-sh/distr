import {Component, input} from '@angular/core';
import {SubscriptionType} from '../types/subscription';

@Component({
  selector: 'app-plan-badge',
  host: {class: 'distr-badge distr-badge-muted capitalize'},
  template: `{{ plan() }}`,
})
export class PlanBadgeComponent {
  public readonly plan = input.required<SubscriptionType>();
}
