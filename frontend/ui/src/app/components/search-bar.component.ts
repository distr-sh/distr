import {afterNextRender, Component, ElementRef, input, viewChild} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faMagnifyingGlass} from '@fortawesome/free-solid-svg-icons';
import {AutotrimDirective} from '../directives/autotrim.directive';

let nextId = 0;

@Component({
  selector: 'app-search-bar',
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective],
  host: {class: 'block w-full md:max-w-md'},
  template: `
    <div class="relative" role="search">
      <div class="absolute inset-y-0 left-0 flex items-center pl-3 pointer-events-none">
        <fa-icon [icon]="faMagnifyingGlass" class="text-gray-500 dark:text-gray-400" />
      </div>
      <label [for]="id" class="sr-only">Search</label>
      <input
        #input
        [id]="id"
        [formControl]="control()"
        [placeholder]="placeholder()"
        autotrim
        type="search"
        class="distr-input p-2 pl-10" />
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

  constructor() {
    afterNextRender(() => {
      if (this.autofocus()) {
        this.input().nativeElement.focus();
      }
    });
  }
}
