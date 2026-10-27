import {Component, inject, input} from '@angular/core';
import {takeUntilDestroyed, toObservable} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faCircleExclamation, faCircleInfo, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, filter, timer} from 'rxjs';
import {ToastService} from '../services/toast.service';

export type ToastType = 'success' | 'error' | 'info';

const autoCloseDuration = 5000;

@Component({
  selector: 'app-toast',
  template: `
    <div
      class="mb-4 flex w-full max-w-xs items-center gap-3 rounded-lg border border-gray-200 bg-white p-4 text-gray-500 shadow-sm dark:border-gray-700 dark:bg-gray-800 dark:text-gray-400"
      role="alert">
      @switch (type()) {
        @case ('error') {
          <fa-icon
            [icon]="faCircleExclamation"
            size="lg"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-red-100 text-red-500 dark:bg-red-800 dark:text-red-200" />
        }
        @case ('success') {
          <fa-icon
            [icon]="faCircleCheck"
            size="lg"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-green-100 text-green-500 dark:bg-green-800 dark:text-green-200" />
        }
        @case ('info') {
          <fa-icon
            [icon]="faCircleInfo"
            size="lg"
            class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-blue-100 text-blue-500 dark:bg-blue-800 dark:text-blue-200" />
        }
      }
      <div class="overflow-hidden text-sm font-normal">{{ message() }}</div>
      <button
        type="button"
        (click)="remove()"
        class="-mx-1.5 -my-1.5 ms-auto inline-flex h-8 w-8 items-center justify-center rounded-lg bg-white p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-900 focus:ring-2 focus:ring-gray-300 dark:bg-gray-800 dark:text-gray-500 dark:hover:bg-gray-700 dark:hover:text-white"
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

  protected readonly faCircleCheck = faCircleCheck;
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
