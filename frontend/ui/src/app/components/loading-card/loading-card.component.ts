import {Component, input} from '@angular/core';

export type LoadingCardLayout = 'row' | 'timeline' | 'list';

@Component({
  selector: 'app-loading-card',
  templateUrl: './loading-card.component.html',
  styleUrl: './loading-card.component.scss',
})
export class LoadingCardComponent {
  /**
   * `row` has the dimensions of one customer home list row, so the card keeps its height once the list arrives.
   * Keep it in line with the layout of `app-customer-home-artifact-item`, the shortest row.
   */
  public readonly layout = input<LoadingCardLayout>('list');

  protected readonly timelineEntries = [0, 1, 2, 3, 4];
}
