import {ChangeDetectionStrategy, Component, computed, effect, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheckCircle} from '@fortawesome/free-solid-svg-icons';
import {map} from 'rxjs';
import {OrganizationService} from '../services/organization.service';

@Component({
  selector: 'app-subscription-callback',
  templateUrl: './subscription-callback.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, RouterLink],
})
export class SubscriptionCallbackComponent {
  private readonly organizationService = inject(OrganizationService);
  private readonly route = inject(ActivatedRoute);

  protected readonly faCheckCircle = faCheckCircle;

  private readonly subscriptionType = toSignal(this.organizationService.get().pipe(map((org) => org.subscriptionType)));
  private readonly pendingPlan = toSignal(this.route.queryParamMap.pipe(map((params) => params.get('pendingPlan'))));

  // Stripe confirms subscription changes asynchronously via webhook. Until then, the
  // organization still has its previous subscription type.
  protected readonly isProcessing = computed(() => {
    const subscriptionType = this.subscriptionType();
    if (subscriptionType === undefined) {
      return undefined;
    }
    return subscriptionType === 'trial' || (!!this.pendingPlan() && subscriptionType !== this.pendingPlan());
  });

  constructor() {
    effect(() => {
      if (this.isProcessing()) {
        setTimeout(() => location.reload(), 5000);
      }
    });
  }
}
