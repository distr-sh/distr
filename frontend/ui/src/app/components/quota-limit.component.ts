import {ChangeDetectionStrategy, Component, computed, input} from '@angular/core';
import {UNLIMITED_QTY} from '../types/subscription';

@Component({
  selector: 'app-quota-limit',
  changeDetection: ChangeDetectionStrategy.Eager,
  template: `
    @let l = limit();
    @let p = percentage();
    @if (l !== undefined && p >= 50) {
      <div class="text-gray-900 dark:text-white flex items-center gap-2 md:w-36">
        <div class="bg-gray-200 rounded-full h-1.5 dark:bg-gray-700 flex-1">
          <div
            class="h-1.5 rounded-full"
            [class.bg-green-600]="!isLimitDanger()"
            [class.bg-blue-600]="isLimitDanger() && !isLimitCritical()"
            [class.bg-yellow-400]="isLimitCritical() && !isLimitReached()"
            [class.bg-red-600]="isLimitReached()"
            [class.dark:bg-red-500]="isLimitReached()"
            [style]="{width: p + '%'}"></div>
        </div>
        <span class="text-sm" [class.text-red-700]="isLimitReached()" [class.dark:text-red-500]="isLimitReached()">
          {{ usage() ?? 0 }}/{{ l }}
        </span>
      </div>
    }
  `,
})
export class QuotaLimitComponent {
  public readonly usage = input<number>();
  public readonly limit = input<number>();
  protected readonly percentage = computed(() => {
    const u = this.usage();
    const l = this.limit();
    if (l === undefined || l === UNLIMITED_QTY) {
      return 0;
    }
    return Math.min(100, Math.round(((u ?? 0) / l) * 100));
  });
  public isLimitDanger = computed(() => this.percentage() >= 75);
  public isLimitCritical = computed(() => this.percentage() >= 85);
  public isLimitReached = computed(() => this.percentage() >= 100);
}
