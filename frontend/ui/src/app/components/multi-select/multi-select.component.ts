import {OverlayModule} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, ElementRef, input, model, signal, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {map} from 'rxjs';
import {SearchBarComponent} from '../search-bar.component';

export interface MultiSelectOption<T extends string = string> {
  value: T;
  label: string;
}

// above this many options a searchable dropdown gets a search field
const SEARCH_THRESHOLD = 5;

/**
 * A dropdown that filters by any number of options at once, matching the services picker in the
 * deployment log viewer. Selecting nothing means "no filter" rather than "match nothing".
 */
@Component({
  selector: 'app-multi-select',
  templateUrl: './multi-select.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  host: {class: 'block'},
  imports: [OverlayModule, FaIconComponent, SearchBarComponent],
})
export class MultiSelectComponent<T extends string = string> {
  public readonly options = input.required<MultiSelectOption<T>[]>();
  public readonly selected = model<T[]>([]);
  public readonly placeholder = input('Select');
  /** Plural noun for the trigger once more than one option is selected, e.g. "severities". */
  public readonly itemLabel = input('items');
  /** Distinguishes the checkbox ids when several instances share a page. */
  public readonly idPrefix = input('multi-select');
  public readonly searchable = input(false);
  public readonly searchPlaceholder = input('Search');

  protected readonly faChevronDown = faChevronDown;

  protected readonly open = signal(false);
  protected width = 0;
  private readonly trigger = viewChild.required<ElementRef<HTMLElement>>('trigger');

  protected readonly search = new FormControl<string>('', {nonNullable: true});
  private readonly searchTerm = toSignal(this.search.valueChanges.pipe(map((term) => term.trim().toLowerCase())), {
    initialValue: '',
  });
  protected readonly isSearchVisible = computed(() => this.searchable() && this.options().length > SEARCH_THRESHOLD);
  protected readonly visibleOptions = computed(() => {
    const term = this.searchTerm();
    return term ? this.options().filter((option) => option.label.toLowerCase().includes(term)) : this.options();
  });

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

  protected readonly allSelected = computed(() => {
    const selected = this.selected();
    const visible = this.visibleOptions();
    return visible.length > 0 && visible.every((option) => selected.includes(option.value));
  });
  protected readonly someSelected = computed(
    () => !this.allSelected() && this.visibleOptions().some((option) => this.selected().includes(option.value))
  );

  protected toggleOpen(): void {
    this.search.setValue('');
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
    const visible = this.visibleOptions().map((option) => option.value);
    this.selected.update((selected) =>
      this.allSelected()
        ? selected.filter((value) => !visible.includes(value))
        : [...selected, ...visible.filter((value) => !selected.includes(value))]
    );
  }
}
