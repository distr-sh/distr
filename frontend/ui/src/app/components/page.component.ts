import {Component, computed, input} from '@angular/core';

export type PageVariant = 'default' | 'full' | 'narrow';

const maxWidths: Record<PageVariant, string> = {
  default: 'max-w-screen-2xl',
  full: 'max-w-none',
  narrow: 'max-w-screen-lg',
};

@Component({
  selector: 'app-page',
  template: `
    <section class="pt-3 pb-32 sm:pt-5 sm:pb-40 antialiased">
      <div class="mx-auto w-full px-4 lg:px-12" [class]="maxWidth()">
        <ng-content />
      </div>
    </section>
  `,
})
export class PageComponent {
  public readonly variant = input<PageVariant>('default');

  protected readonly maxWidth = computed(() => maxWidths[this.variant()]);
}
