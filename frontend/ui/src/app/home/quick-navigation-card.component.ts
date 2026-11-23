import {CdkOverlayOrigin} from '@angular/cdk/overlay';
import {LowerCasePipe, NgTemplateOutlet, TitleCasePipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {Params, RouterLink} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faCircleExclamation, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../components/card-heading.component';
import {PlanBadgeComponent} from '../components/plan-badge.component';
import {PlanHintOverlayComponent} from '../components/plan-hint-overlay.component';
import {FeatureLock, upgradePlan} from '../types/feature-lock';

export type QuickNavigationCardType = 'neutral' | 'success' | 'warning' | 'error';

/**
 * Shows `count` as the headline, labeled by the projected content, above the projected `[subtitle]`, as a link to
 * `link` if one is given. Without `count`, the card shows no headline.
 * Every type but `neutral` colors the card and replaces `icon` with an icon of its own. With `locked`, the card
 * shows a plan badge and opens a plan hint instead of following `link`, like a locked sidebar entry.
 */
@Component({
  selector: 'app-quick-navigation-card',
  templateUrl: './quick-navigation-card.component.html',
  styleUrl: './quick-navigation-card.component.scss',
  imports: [
    NgTemplateOutlet,
    RouterLink,
    FaIconComponent,
    CardHeadingComponent,
    CdkOverlayOrigin,
    PlanBadgeComponent,
    PlanHintOverlayComponent,
    LowerCasePipe,
    TitleCasePipe,
  ],
  host: {
    class: 'block',
    '[class]': 'typeClass()',
  },
})
export class QuickNavigationCardComponent {
  public readonly icon = input.required<IconDefinition>();
  public readonly heading = input.required<string>();
  public readonly type = input<QuickNavigationCardType>('neutral');
  public readonly count = input<number>();
  public readonly link = input<string | readonly unknown[]>();
  public readonly queryParams = input<Params>();
  public readonly locked = input<FeatureLock>();

  protected readonly lockedPlan = computed(() => {
    const locked = this.locked();
    return locked && upgradePlan(locked);
  });
  protected readonly typeClass = computed(() => `card-${this.type()}`);
  protected readonly alertClass = computed(() => {
    const type = this.type();
    return type === 'neutral' ? undefined : `distr-alert-${type}`;
  });
  protected readonly typeIcon = computed(() => {
    switch (this.type()) {
      case 'success':
        return faCircleCheck;
      case 'warning':
        return faTriangleExclamation;
      case 'error':
        return faCircleExclamation;
      case 'neutral':
        return undefined;
    }
  });
}
