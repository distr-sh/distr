import {ChangeDetectionStrategy, Component, effect, input, output, signal} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faPen} from '@fortawesome/free-solid-svg-icons';
import {AutofocusDirective} from '../directives/autofocus.directive';
import {AutotrimDirective} from '../directives/autotrim.directive';

export type InlineEditSize = 'sm' | 'lg';
export type InlineEditDisplayTag = 'span' | 'h3';

@Component({
  selector: 'app-inline-edit',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, AutofocusDirective],
  template: `
    @if (editing()) {
      <form class="flex" [formGroup]="form" (ngSubmit)="submit()">
        <input
          formControlName="value"
          autotrim
          autofocus
          type="text"
          [placeholder]="placeholder()"
          (keydown.escape)="cancel()"
          (blur)="cancel()"
          class="distr-input rounded-none rounded-s-lg"
          [class]="size() === 'lg' ? 'w-48' : 'w-32'" />
        <button
          type="submit"
          [disabled]="loading()"
          (mousedown)="$event.preventDefault()"
          class="distr-btn-primary text-white rounded-none rounded-e-lg"
          [class]="size() === 'lg' ? 'px-4' : 'px-3'">
          <fa-icon [icon]="faCheck" />
        </button>
      </form>
    } @else {
      <span class="inline-flex items-center">
        @if (value()) {
          @if (tag() === 'h3') {
            <h3 [class]="textClass()" [title]="value()">{{ value() }}</h3>
          } @else {
            <span [class]="textClass()" [title]="value()">{{ value() }}</span>
          }
        } @else if (emptyLabel()) {
          <span class="text-gray-500">{{ emptyLabel() }}</span>
        }
        @if (editable()) {
          <button type="button" aria-label="Edit" (click)="enable()" class="text-gray-900 dark:text-gray-400 ms-2">
            <fa-icon [icon]="faPen" />
          </button>
        }
      </span>
    }
  `,
})
export class InlineEditComponent {
  readonly value = input.required<string>();
  readonly placeholder = input('');
  readonly editable = input(true);
  readonly required = input(true);
  readonly loading = input(false);
  readonly textClass = input('');
  readonly emptyLabel = input('');
  readonly size = input<InlineEditSize>('sm');
  readonly tag = input<InlineEditDisplayTag>('span');

  readonly save = output<string>();

  protected readonly editing = signal(false);
  protected readonly form = new FormGroup({
    value: new FormControl('', {nonNullable: true}),
  });

  protected readonly faCheck = faCheck;
  protected readonly faPen = faPen;

  constructor() {
    effect(() => {
      this.form.controls.value.setValidators(this.required() ? [Validators.required] : []);
      this.form.controls.value.updateValueAndValidity({emitEvent: false});
    });
    // Exit edit mode when the bound value changes externally (e.g. navigating to another
    // entity on the same route, or a successful save) so a pending edit can never be saved
    // against the wrong entity.
    effect(() => {
      this.value();
      this.editing.set(false);
    });
  }

  protected enable() {
    this.form.reset({value: this.value()});
    this.editing.set(true);
  }

  protected cancel() {
    this.editing.set(false);
  }

  protected submit() {
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      return;
    }
    const newValue = this.form.controls.value.value.trim();
    if (newValue === this.value()) {
      this.editing.set(false);
      return;
    }
    this.save.emit(newValue);
  }
}
