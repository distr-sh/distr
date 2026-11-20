import {Component, computed, input} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {from} from 'rxjs';
import {WEBSITE_URL} from '../../constants';
import {environment} from '../../env/env';
import {getRemoteEnvironment} from '../../env/remote';
import {canOpenSubscription} from '../organization-settings/subscription/subscription.guard';
import {SubscriptionType} from '../types/subscription';
import {PlanBadgeComponent} from './plan-badge.component';

@Component({
  selector: 'app-plan-feature-hint',
  imports: [PlanBadgeComponent, RouterLink],
  templateUrl: './plan-feature-hint.component.html',
})
export class PlanFeatureHintComponent {
  public readonly plan = input.required<SubscriptionType>();

  protected readonly websiteUrl = WEBSITE_URL;
  protected readonly edition = environment.edition;
  private readonly remoteEnvironment = toSignal(from(getRemoteEnvironment()));
  protected readonly licenseEnforced = computed(() => this.remoteEnvironment()?.licenseEnforced ?? false);
  protected readonly canOpenSubscription = toSignal(canOpenSubscription(), {initialValue: false});
}
