import {ChangeDetectionStrategy, Component, computed, input} from '@angular/core';
import {UNLIMITED_QTY} from '../types/subscription';

@Component({
  selector: 'app-quota-limit',
  changeDetection: ChangeDetectionStrategy.Eager,
  template: `
    @if (shouldShow()) {
      <span
        class="text-sm"
        [class]="
          isLimitReached()
            ? ['text-red-700', 'dark:text-red-500']
            : isLimitAlmostReached()
              ? ['text-yellow-800', 'dark:text-yellow-300']
              : ['text-gray-500', 'dark:text-gray-400']
        ">
        {{ remainingCount() }}{{ label() ? ' ' + label() : '' }} remaining
      </span>
    }
  `,
})
export class QuotaLimitComponent {
  public readonly usage = input<number>();
  public readonly limit = input<number>();
  public readonly label = input<string>('');

  protected readonly remainingCount = computed(() => {
    const u = this.usage() ?? 0;
    const l = this.limit();
    if (l === undefined || l === UNLIMITED_QTY) {
      return undefined;
    }
    return Math.max(0, l - u);
  });

  protected readonly shouldShow = computed(() => {
    const l = this.limit();
    const r = this.remainingCount();
    if (l === undefined || l === UNLIMITED_QTY || l === 0 || r === undefined) {
      return false;
    }
    return r / l <= 0.5 || r <= 3;
  });

  public isLimitAlmostReached = computed(() => this.remainingCount() === 1);

  public isLimitReached = computed(() => this.remainingCount() === 0);
}
