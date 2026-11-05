import {Component, computed, inject, input} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';

export type PageVariant = 'default' | 'full' | 'narrow';

const maxWidths: Record<PageVariant, string> = {
  default: 'max-w-screen-2xl',
  full: 'max-w-none',
  narrow: 'max-w-screen-lg',
};

@Component({
  selector: 'app-page',
  template: `
    <section class="pt-3 pb-32 antialiased sm:pt-5 sm:pb-40">
      <div class="mx-auto w-full px-4 lg:px-12" [class]="maxWidth()">
        @if (heading(); as heading) {
          <h2
            class="mb-4 text-3xl leading-none font-extrabold tracking-tight text-gray-900 md:text-4xl dark:text-white">
            {{ heading }}
          </h2>
        }
        <ng-content />
      </div>
    </section>
  `,
})
export class PageComponent {
  public readonly variant = input<PageVariant>('default');

  protected readonly maxWidth = computed(() => maxWidths[this.variant()]);
  protected readonly heading = toSignal(inject(ActivatedRoute).title);
}
