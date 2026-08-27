import {OverlayModule} from '@angular/cdk/overlay';
import {NgClass} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, input, output, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown} from '@fortawesome/free-solid-svg-icons';

export interface BadgeSelectOption<T extends string = string> {
  value: T;
  label: string;
  /** The colors of the badge, in the shape the `distr-status-badge` helpers return. */
  badgeClass: string;
}

/**
 * A status badge that is also the control changing it, so that the value and the way to set it
 * are the same element wherever a state is shown.
 *
 * Selecting an option only emits: the caller writes the value through the API and passes back
 * what the server returned, which keeps the badge from showing a change that failed.
 */
@Component({
  selector: 'app-badge-select',
  templateUrl: './badge-select.component.html',
  host: {class: 'inline-block'},
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [OverlayModule, NgClass, FaIconComponent],
})
export class BadgeSelectComponent<T extends string = string> {
  public readonly options = input.required<BadgeSelectOption<T>[]>();
  public readonly value = input.required<T>();
  public readonly disabled = input(false);
  public readonly ariaLabel = input('Change');

  public readonly selected = output<T>();

  protected readonly faChevronDown = faChevronDown;
  protected readonly open = signal(false);

  protected readonly current = computed(() => this.options().find((option) => option.value === this.value()));

  // Stops the click from reaching an enclosing row link, which is what the list rows are.
  protected toggleOpen(event: Event): void {
    event.stopPropagation();
    this.open.update((open) => !open);
  }

  protected select(event: Event, value: T): void {
    event.stopPropagation();
    this.open.set(false);
    if (value !== this.value()) {
      this.selected.emit(value);
    }
  }
}
