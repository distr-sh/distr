import {Component, computed, input} from '@angular/core';

/**
 * Decides how wide the content of a page may grow:
 * - `full` for content that uses the whole viewport, e.g. the dashboard
 * - `panel` for a single `distr-panel` holding prose or a form, e.g. the organization settings
 * - `table` for one or more `distr-table-panel`
 * - `cards` for a `distr-card-grid`
 */
export type PageVariant = 'full' | 'panel' | 'table' | 'cards';

const maxWidths: Record<PageVariant, string> = {
  full: 'max-w-none',
  panel: 'max-w-screen-lg',
  table: 'max-w-screen-2xl',
  cards: 'max-w-screen-2xl',
};

@Component({
  selector: 'app-page',
  template: `
    <section class="py-3 sm:py-5 antialiased">
      <div class="mx-auto w-full px-4 lg:px-12" [class]="maxWidth()">
        <ng-content />
      </div>
    </section>
  `,
})
export class PageComponent {
  public readonly variant = input.required<PageVariant>();

  protected readonly maxWidth = computed(() => maxWidths[this.variant()]);
}
