import {OverlayModule} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, ElementRef, input, model, signal, viewChild} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown} from '@fortawesome/free-solid-svg-icons';

export interface MultiSelectOption<T extends string = string> {
  value: T;
  label: string;
}

/**
 * A dropdown that filters by any number of options at once, matching the services picker in the
 * deployment log viewer. Selecting nothing means "no filter" rather than "match nothing".
 */
@Component({
  selector: 'app-multi-select',
  templateUrl: './multi-select.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  host: {class: 'block'},
  imports: [OverlayModule, FaIconComponent],
})
export class MultiSelectComponent<T extends string = string> {
  public readonly options = input.required<MultiSelectOption<T>[]>();
  public readonly selected = model<T[]>([]);
  public readonly placeholder = input('Select');
  /** Plural noun for the trigger once more than one option is selected, e.g. "severities". */
  public readonly itemLabel = input('items');
  /** Distinguishes the checkbox ids when several instances share a page. */
  public readonly idPrefix = input('multi-select');

  protected readonly faChevronDown = faChevronDown;

  protected readonly open = signal(false);
  protected width = 0;
  private readonly trigger = viewChild.required<ElementRef<HTMLElement>>('trigger');

  protected readonly triggerLabel = computed(() => {
    const selected = this.selected();
    if (selected.length === 0) {
      return this.placeholder();
    }
    if (selected.length === 1) {
      return this.options().find((option) => option.value === selected[0])?.label ?? selected[0];
    }
    return `${selected.length} ${this.itemLabel()} selected`;
  });

  protected readonly allSelected = computed(
    () => this.options().length > 0 && this.selected().length === this.options().length
  );
  protected readonly someSelected = computed(() => this.selected().length > 0 && !this.allSelected());

  protected toggleOpen(): void {
    this.open.update((open) => !open);
    if (this.open()) {
      this.width = this.trigger().nativeElement.getBoundingClientRect().width;
    }
  }

  protected isSelected(value: T): boolean {
    return this.selected().includes(value);
  }

  protected toggle(value: T): void {
    this.selected.update((selected) =>
      selected.includes(value) ? selected.filter((v) => v !== value) : [...selected, value]
    );
  }

  protected toggleAll(): void {
    this.selected.set(this.allSelected() ? [] : this.options().map((option) => option.value));
  }
}
