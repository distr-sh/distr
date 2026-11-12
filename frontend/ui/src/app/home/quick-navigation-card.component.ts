import {NgTemplateOutlet} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {Params, RouterLink} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faCircleExclamation, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../components/card-heading.component';

export type QuickNavigationCardType = 'neutral' | 'success' | 'warning' | 'error';

/**
 * Shows the projected content as the headline above the projected `[subtitle]`, as a link to `link` if one is given.
 * Every type but `neutral` colors the card and replaces `icon` with an icon of its own.
 */
@Component({
  selector: 'app-quick-navigation-card',
  templateUrl: './quick-navigation-card.component.html',
  styleUrl: './quick-navigation-card.component.scss',
  imports: [NgTemplateOutlet, RouterLink, FaIconComponent, CardHeadingComponent],
  host: {
    class: 'block',
    '[attr.data-type]': 'type()',
  },
})
export class QuickNavigationCardComponent {
  public readonly icon = input.required<IconDefinition>();
  public readonly heading = input.required<string>();
  public readonly type = input<QuickNavigationCardType>('neutral');
  public readonly link = input<string | readonly unknown[]>();
  public readonly queryParams = input<Params>();

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
