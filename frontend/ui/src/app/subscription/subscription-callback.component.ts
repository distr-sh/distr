import {ChangeDetectionStrategy, Component, effect, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
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

  protected readonly faCheckCircle = faCheckCircle;

  protected readonly isTrial = toSignal(
    this.organizationService.get().pipe(map((org) => org.subscriptionType === 'trial'))
  );

  constructor() {
    effect(() => {
      if (this.isTrial()) {
        setTimeout(() => location.reload(), 5000);
      }
    });
  }
}
