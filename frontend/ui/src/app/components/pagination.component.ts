import {Component, computed, input, model} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronLeft, faChevronRight} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-pagination',
  imports: [FaIconComponent],
  host: {
    class:
      'flex items-center justify-between gap-3 px-6 py-3 border-t border-gray-200 dark:border-gray-700 ' +
      'text-sm text-gray-500 dark:text-gray-400',
  },
  template: `
    <p aria-live="polite">
      Showing
      <span class="font-medium text-gray-900 dark:text-white">{{ firstItem() }}</span>
      to
      <span class="font-medium text-gray-900 dark:text-white">{{ lastItem() }}</span>
      of
      <span class="font-medium text-gray-900 dark:text-white">{{ total() }}</span>
    </p>
    <nav class="flex items-center gap-3" aria-label="Pagination">
      <button
        type="button"
        class="distr-btn-secondary  py-2 px-3 "
        [disabled]="page() === 0"
        (click)="goTo(page() - 1)">
        <fa-icon [icon]="faChevronLeft" />
      </button>
      <span>
        Page
        <span class="font-medium text-gray-900 dark:text-white">{{ page() + 1 }}</span>
        of
        <span class="font-medium text-gray-900 dark:text-white">{{ pageCount() }}</span>
      </span>
      <button
        type="button"
        class="distr-btn-secondary  py-2 px-3 "
        [disabled]="page() >= pageCount() - 1"
        (click)="goTo(page() + 1)">
        <fa-icon [icon]="faChevronRight" />
      </button>
    </nav>
  `,
})
export class PaginationComponent {
  public readonly total = input.required<number>();
  public readonly pageSize = input.required<number>();
  /** Index of the page that is shown, counting from zero. */
  public readonly page = model.required<number>();

  protected readonly pageCount = computed(() => Math.max(1, Math.ceil(this.total() / this.pageSize())));
  protected readonly firstItem = computed(() => (this.total() === 0 ? 0 : this.page() * this.pageSize() + 1));
  protected readonly lastItem = computed(() => Math.min(this.total(), (this.page() + 1) * this.pageSize()));

  protected readonly faChevronLeft = faChevronLeft;
  protected readonly faChevronRight = faChevronRight;

  protected goTo(page: number) {
    this.page.set(Math.max(0, Math.min(page, this.pageCount() - 1)));
  }
}
