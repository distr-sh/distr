import {NgTemplateOutlet} from '@angular/common';
import {Component, input} from '@angular/core';
import {Params, RouterLink} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faCircleExclamation} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../components/card-heading.component';

export type QuickNavigationCardType = 'default' | 'error';

/**
 * Shows `count` with the projected `[attention]` subtitle as a link to `link` while it is above zero, and the projected
 * `[allGood]` message otherwise, as a link to `allGoodLink` if one is given.
 */
@Component({
  selector: 'app-quick-navigation-card',
  templateUrl: './quick-navigation-card.component.html',
  styleUrl: './quick-navigation-card.component.scss',
  imports: [NgTemplateOutlet, RouterLink, FaIconComponent, CardHeadingComponent],
  host: {class: 'block'},
})
export class QuickNavigationCardComponent {
  public readonly icon = input.required<IconDefinition>();
  public readonly heading = input.required<string>();
  public readonly count = input.required<number>();
  public readonly link = input.required<string | readonly unknown[]>();
  public readonly queryParams = input<Params>();
  public readonly allGoodLink = input<string | readonly unknown[]>();
  public readonly type = input<QuickNavigationCardType>('default');

  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faCircleExclamation = faCircleExclamation;
}
