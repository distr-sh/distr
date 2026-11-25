import {Component, computed, effect, inject, input, output, signal} from '@angular/core';
import {FormBuilder, FormsModule, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faCircleExclamation, faRotate, faTrash} from '@fortawesome/free-solid-svg-icons';
import {RelativeDatePipe} from '../../../../util/dates';
import {HOSTNAME_MAX_LENGTH, HOSTNAME_REGEX} from '../../../../util/validation';
import {AutotrimDirective} from '../../../directives/autotrim.directive';
import {CustomDomain} from '../../../types/custom-domain';
import {ClipComponent} from '../../clip.component';

@Component({
  selector: 'app-domain-field',
  templateUrl: './domain-field.component.html',
  imports: [FaIconComponent, FormsModule, ReactiveFormsModule, AutotrimDirective, ClipComponent, RelativeDatePipe],
})
export class DomainFieldComponent {
  protected readonly faTrash = faTrash;
  protected readonly faRotate = faRotate;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faCircleExclamation = faCircleExclamation;

  private readonly fb = inject(FormBuilder).nonNullable;

  public readonly fieldId = input.required<string>();
  public readonly label = input.required<string>();
  public readonly placeholder = input.required<string>();
  public readonly invalidHint = input.required<string>();
  public readonly removeAriaLabel = input.required<string>();
  public readonly domain = input<CustomDomain>();
  public readonly cnameTarget = input<string>();
  public readonly saving = input(false);
  public readonly verifying = input(false);
  public readonly optionalCheckboxLabel = input<string>();

  public readonly save = output<string>();
  public readonly remove = output<void>();
  public readonly recheck = output<void>();

  protected readonly checkboxId = computed(() => `${this.fieldId()}Enabled`);
  protected readonly enabled = signal(false);
  protected readonly expanded = computed(() => !this.optionalCheckboxLabel() || !!this.domain() || this.enabled());

  protected readonly control = this.fb.control('', [
    Validators.required,
    Validators.pattern(HOSTNAME_REGEX),
    Validators.maxLength(HOSTNAME_MAX_LENGTH),
  ]);

  protected malformed(): boolean {
    return this.control.hasError('pattern') || this.control.hasError('maxlength');
  }

  constructor() {
    // Clear only once saved, so the typed value survives a failed save.
    effect(() => {
      if (this.domain()) {
        this.control.reset('', {emitEvent: false});
      }
    });
  }

  protected submit(event: Event) {
    event.preventDefault();
    this.control.markAsTouched();
    if (this.control.invalid) {
      return;
    }
    this.save.emit(this.control.value.toLowerCase());
  }
}
