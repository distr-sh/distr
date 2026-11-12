import {NgTemplateOutlet} from '@angular/common';
import {Component, input} from '@angular/core';
import {Params, RouterLink} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faCircleExclamation, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../components/card-heading.component';

export type QuickNavigationCardType = 'default' | 'warning' | 'error';

/**
 * Shows `count`, followed by the projected `[countLabel]`, with the projected `[attention]` subtitle as a link to
 * `link` while it is above zero. Otherwise it shows the projected `[allGood]` message above the `[allGoodSubtitle]`,
 * or the projected `[empty]` message with `isEmpty`, as a link to `overviewLink` if one is given.
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
  public readonly isEmpty = input(false);
  public readonly link = input.required<string | readonly unknown[]>();
  public readonly queryParams = input<Params>();
  public readonly overviewLink = input<string | readonly unknown[]>();
  public readonly type = input<QuickNavigationCardType>('default');

  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faTriangleExclamation = faTriangleExclamation;
}
