import {ChangeDetectionStrategy, Component, effect, ElementRef, inject, input, signal, viewChild} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faPen} from '@fortawesome/free-solid-svg-icons';
import {lastValueFrom, Observable} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ToastService} from '../services/toast.service';

export type InlineEditSize = 'sm' | 'lg';
export type InlineEditDisplayTag = 'span' | 'h3';

@Component({
  selector: 'app-inline-edit',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective],
  template: `
    @if (editing()) {
      <form class="flex" [formGroup]="form" (ngSubmit)="submit()">
        <input
          formControlName="value"
          autotrim
          #valueInput
          type="text"
          [placeholder]="placeholder()"
          class="distr-input rounded-none rounded-s-lg"
          [class]="size() === 'lg' ? 'w-48' : 'w-32'" />
        <button
          type="submit"
          [disabled]="loading()"
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
  private readonly toast = inject(ToastService);

  readonly value = input.required<string>();
  readonly onSave = input.required<(value: string) => Observable<unknown>>();
  readonly placeholder = input('');
  readonly editable = input(true);
  readonly required = input(true);
  readonly textClass = input('');
  readonly emptyLabel = input('');
  readonly size = input<InlineEditSize>('sm');
  readonly tag = input<InlineEditDisplayTag>('span');

  protected readonly editing = signal(false);
  protected readonly loading = signal(false);
  protected readonly form = new FormGroup({
    value: new FormControl('', {nonNullable: true}),
  });
  private readonly valueInput = viewChild<ElementRef<HTMLInputElement>>('valueInput');

  protected readonly faCheck = faCheck;
  protected readonly faPen = faPen;

  constructor() {
    effect(() => {
      this.form.controls.value.setValidators(this.required() ? [Validators.required] : []);
      this.form.controls.value.updateValueAndValidity({emitEvent: false});
    });
  }

  protected enable() {
    this.form.reset({value: this.value()});
    this.editing.set(true);
    setTimeout(() => this.valueInput()?.nativeElement.focus(), 10);
  }

  protected async submit() {
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      return;
    }
    this.loading.set(true);
    try {
      await lastValueFrom(this.onSave()(this.form.controls.value.value.trim()));
      this.editing.set(false);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.loading.set(false);
    }
  }
}
