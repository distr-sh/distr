import {Component, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faCircleExclamation, faCircleInfo, faXmark} from '@fortawesome/free-solid-svg-icons';
import {ToastNoAnimation} from 'ngx-toastr';
import {firstValueFrom, Subject} from 'rxjs';

export type ToastType = 'success' | 'error' | 'info';

@Component({
  styles: [
    `
      :host {
        pointer-events: all;
        background-color: transparent;
        background-image: none;
      }
    `,
  ],
  template: `
    @if (!isDismissed()) {
      <div
        [class.border-red-300]="options().payload === 'error'"
        [class.dark:border-red-800]="options().payload === 'error'"
        [class.border-green-300]="options().payload === 'success'"
        [class.dark:border-green-800]="options().payload === 'success'"
        [class.border-blue-300]="options().payload === 'info'"
        [class.dark:border-blue-800]="options().payload === 'info'"
        class="flex items-center w-full max-w-xs gap-3 p-4 mb-4 text-gray-500 bg-white rounded-lg shadow-sm dark:text-gray-400 dark:bg-gray-800 border border-gray-200 dark:border-gray-600"
        role="alert"
        animate.enter="animate-fly-skew-in-right"
        animate.leave="animate-fly-out-right"
        (animationend)="onAnimationComplete()">
        @switch (options().payload) {
          @case ('error') {
            <fa-icon
              [icon]="faCircleExclamation"
              size="lg"
              class="inline-flex items-center justify-center shrink-0 w-8 h-8 rounded-lg text-red-500 dark:bg-red-800 bg-red-100 dark:text-red-200" />
          }
          @case ('success') {
            <fa-icon
              [icon]="faCheck"
              size="lg"
              class="inline-flex items-center justify-center shrink-0 w-8 h-8 rounded-lg text-green-500 dark:text-green-800" />
          }
          @case ('info') {
            <fa-icon
              [icon]="faCircleInfo"
              size="lg"
              class="inline-flex items-center justify-center shrink-0 w-8 h-8 rounded-lg text-blue-500 dark:bg-blue-800 bg-blue-100 dark:text-blue-200" />
          }
        }
        <div class="text-sm font-normal overflow-hidden">
          @if (message()) {
            {{ title() }}: {{ message() }}
          } @else {
            {{ title() }}
          }
        </div>
        <button
          type="button"
          (click)="remove()"
          class="ms-auto -mx-1.5 -my-1.5 bg-white text-gray-400 hover:text-gray-900 rounded-lg focus:ring-2 focus:ring-gray-300 p-1.5 hover:bg-gray-100 inline-flex items-center justify-center h-8 w-8 dark:text-gray-500 dark:hover:text-white dark:bg-gray-800 dark:hover:bg-gray-700"
          aria-label="Close">
          <span class="sr-only">Close</span>
          <fa-icon [icon]="faXmark" />
        </button>
      </div>
    }
  `,
  imports: [FaIconComponent],
})
export class ToastComponent extends ToastNoAnimation<ToastType> {
  protected readonly faCheck = faCheck;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faCircleInfo = faCircleInfo;
  protected readonly faXmark = faXmark;

  protected readonly isDismissed = signal(false);
  private readonly animationComplete$ = new Subject<void>();

  override async remove() {
    this.isDismissed.set(true);
    await firstValueFrom(this.animationComplete$);
    super.remove();
  }

  protected onAnimationComplete() {
    this.animationComplete$.next();
  }
}
