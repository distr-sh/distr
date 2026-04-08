import {Component, inject, input} from '@angular/core';
import {takeUntilDestroyed, toObservable} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faCircleExclamation, faCircleInfo, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, filter, timer} from 'rxjs';
import {ToastService} from '../services/toast.service';

export type ToastType = 'success' | 'error' | 'info';

const autoCloseDuration = 5000;

@Component({
  selector: 'app-toast',
  template: `
    <div
      [class.border-red-300]="type() === 'error'"
      [class.dark:border-red-800]="type() === 'error'"
      [class.border-green-300]="type() === 'success'"
      [class.dark:border-green-800]="type() === 'success'"
      [class.border-blue-300]="type() === 'info'"
      [class.dark:border-blue-800]="type() === 'info'"
      class="flex items-center w-full max-w-xs gap-3 p-4 mb-4 text-gray-500 bg-white rounded-lg shadow-sm dark:text-gray-400 dark:bg-gray-800 border border-gray-200 dark:border-gray-600"
      role="alert">
      @switch (type()) {
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
      <div class="text-sm font-normal overflow-hidden">{{ message() }}</div>
      <button
        type="button"
        (click)="remove()"
        class="ms-auto -mx-1.5 -my-1.5 bg-white text-gray-400 hover:text-gray-900 rounded-lg focus:ring-2 focus:ring-gray-300 p-1.5 hover:bg-gray-100 inline-flex items-center justify-center h-8 w-8 dark:text-gray-500 dark:hover:text-white dark:bg-gray-800 dark:hover:bg-gray-700"
        aria-label="Close">
        <span class="sr-only">Close</span>
        <fa-icon [icon]="faXmark" />
      </button>
    </div>
  `,
  imports: [FaIconComponent],
})
export class ToastComponent {
  private readonly toastService = inject(ToastService);

  readonly id = input.required<string>();
  readonly type = input.required<ToastType>();
  readonly message = input.required<string>();
  readonly autoClose = input(false);

  protected readonly faCheck = faCheck;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faCircleInfo = faCircleInfo;
  protected readonly faXmark = faXmark;

  constructor() {
    combineLatest([timer(autoCloseDuration), toObservable(this.autoClose)])
      .pipe(
        filter(([_, autoClose]) => autoClose),
        takeUntilDestroyed()
      )
      .subscribe(() => this.remove());
  }

  protected remove() {
    this.toastService.dismiss(this.id());
  }
}
