import {ChangeDetectionStrategy, Component, computed, effect, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck} from '@fortawesome/free-solid-svg-icons';
import {map} from 'rxjs';
import {PageComponent} from '../components/page.component';
import {SubscriptionService} from '../services/subscription.service';

@Component({
  selector: 'app-subscription-callback',
  templateUrl: './subscription-callback.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, RouterLink, PageComponent],
})
export class SubscriptionCallbackComponent {
  private readonly subscriptionService = inject(SubscriptionService);
  private readonly route = inject(ActivatedRoute);

  protected readonly faCircleCheck = faCircleCheck;

  private readonly subscriptionInfo = toSignal(this.subscriptionService.get());
  private readonly pendingPlan = toSignal(this.route.queryParamMap.pipe(map((params) => params.get('pendingPlan'))));

  // Stripe confirms subscription changes asynchronously via webhook. Until then, the
  // organization still has its previous subscription.
  protected readonly isProcessing = computed(() => {
    const info = this.subscriptionInfo();
    if (info === undefined) {
      return undefined;
    }
    return !info.hasStripeSubscription || (!!this.pendingPlan() && info.subscriptionType !== this.pendingPlan());
  });

  constructor() {
    effect(() => {
      if (this.isProcessing()) {
        setTimeout(() => location.reload(), 5000);
      }
    });
  }
}
