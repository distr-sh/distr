import {CdkConnectedOverlay, CdkOverlayOrigin} from '@angular/cdk/overlay';
import {LowerCasePipe, NgTemplateOutlet, TitleCasePipe} from '@angular/common';
import {Component, input, signal} from '@angular/core';
import {Params, RouterLink} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faCircleExclamation, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../components/card-heading.component';
import {PlanBadgeComponent} from '../components/plan-badge.component';
import {PlanFeatureHintComponent} from '../components/plan-feature-hint.component';
import {UpgradePlan} from '../types/subscription';

export type QuickNavigationCardType = 'neutral' | 'success' | 'warning' | 'error';

/**
 * Shows the projected content as the headline above the projected `[subtitle]`, as a link to `link` if one is given.
 * Every type but `neutral` shows a status badge in the heading, labeled `errorLabel` for `error`. With `lockedPlan`,
 * the card shows a plan badge instead and opens a plan hint instead of following `link`, like a locked sidebar entry.
 */
@Component({
  selector: 'app-quick-navigation-card',
  templateUrl: './quick-navigation-card.component.html',
  imports: [
    NgTemplateOutlet,
    RouterLink,
    FaIconComponent,
    CardHeadingComponent,
    CdkOverlayOrigin,
    CdkConnectedOverlay,
    PlanBadgeComponent,
    PlanFeatureHintComponent,
    LowerCasePipe,
    TitleCasePipe,
  ],
  host: {
    class: 'block',
  },
})
export class QuickNavigationCardComponent {
  public readonly icon = input.required<IconDefinition>();
  public readonly heading = input.required<string>();
  public readonly type = input<QuickNavigationCardType>('neutral');
  public readonly errorLabel = input('Error');
  public readonly link = input<string | readonly unknown[]>();
  public readonly queryParams = input<Params>();
  public readonly lockedPlan = input<UpgradePlan>();

  protected readonly isPlanHintOpen = signal(false);

  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faTriangleExclamation = faTriangleExclamation;
  protected readonly faCircleExclamation = faCircleExclamation;
}
