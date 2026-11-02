import {Component, ElementRef, input, viewChild} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faMagnifyingGlass} from '@fortawesome/free-solid-svg-icons';
import {AutofocusDirective} from '../directives/autofocus.directive';
import {AutotrimDirective} from '../directives/autotrim.directive';

let nextId = 0;

function isEditable(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
  );
}

@Component({
  selector: 'app-search-bar',
  imports: [ReactiveFormsModule, FaIconComponent, AutofocusDirective, AutotrimDirective],
  host: {
    class: 'block w-full md:max-w-md',
    '(document:keydown)': 'onDocumentKeydown($event)',
  },
  template: `
    <div class="relative" role="search">
      <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
        <fa-icon [icon]="faMagnifyingGlass" class="text-gray-500 dark:text-gray-400" />
      </div>
      <label [for]="id" class="sr-only">Search</label>
      <input
        #input
        [id]="id"
        [formControl]="control()"
        [placeholder]="placeholder()"
        [autofocus]="autofocus()"
        aria-keyshortcuts="/"
        autotrim
        type="search"
        class="peer distr-input p-2 pl-10 placeholder-shown:pr-10" />
      <div
        class="pointer-events-none absolute inset-y-0 right-0 hidden items-center pr-3 peer-placeholder-shown:flex peer-focus:hidden">
        <kbd
          class="rounded-sm border border-gray-200 bg-gray-100 px-1.5 font-sans text-xs text-gray-500 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-400">
          /
        </kbd>
      </div>
    </div>
  `,
})
export class SearchBarComponent {
  public readonly control = input.required<FormControl>();
  public readonly placeholder = input.required<string>();
  public readonly autofocus = input(false);

  private readonly input = viewChild.required<ElementRef<HTMLInputElement>>('input');

  // the label needs an id to point at, and more than one search bar can be on the page at a time
  protected readonly id = `search-bar-${nextId++}`;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;

  /** Focuses the first visible search bar of the page when "/" is pressed outside of a text field. */
  protected onDocumentKeydown(event: KeyboardEvent): void {
    const input = this.input().nativeElement;
    if (
      event.defaultPrevented ||
      event.key !== '/' ||
      event.ctrlKey ||
      event.metaKey ||
      event.altKey ||
      isEditable(event.target) ||
      !input.checkVisibility()
    ) {
      return;
    }
    // Also tells the other search bars on the page, whose listeners run after this one, that the key
    // is handled, and keeps the "/" from being typed into the input that just got focus.
    event.preventDefault();
    input.focus();
  }
}
