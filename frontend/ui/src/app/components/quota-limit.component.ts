import {Component, computed, input} from '@angular/core';
import {UNLIMITED_QTY} from '../types/subscription';

@Component({
  selector: 'app-quota-limit',
  template: `
    @if (shouldShow()) {
      @if (isLimitReached()) {
        <span class="text-sm text-red-800 dark:text-red-400">{{ text() }}</span>
      } @else if (isLimitAlmostReached()) {
        <span class="text-sm text-yellow-800 dark:text-yellow-300">{{ text() }}</span>
      } @else {
        <span class="text-sm text-gray-500 dark:text-gray-400">{{ text() }}</span>
      }
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

  protected readonly text = computed(() => {
    const label = this.label();
    return `${this.remainingCount()}${label ? ' ' + label : ''} remaining`;
  });

  public isLimitAlmostReached = computed(() => this.remainingCount() === 1);

  public isLimitReached = computed(() => this.remainingCount() === 0);
}
