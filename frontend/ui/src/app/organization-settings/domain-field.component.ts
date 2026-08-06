import {ChangeDetectionStrategy, Component, inject, input, output} from '@angular/core';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faTrash} from '@fortawesome/free-solid-svg-icons';
import {HOSTNAME_MAX_LENGTH, HOSTNAME_REGEX} from '../../util/validation';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {CustomDomain} from '../types/custom-domain';

// A single CNAME-configurable domain field: shows the existing domain with a remove button once
// configured, or a validated hostname input beforehand. Purely presentational — the domain list and
// the actual create/delete calls belong to the parent, which knows the domain type and scope.
@Component({
  selector: 'app-domain-field',
  templateUrl: './domain-field.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective],
})
export class DomainFieldComponent {
  protected readonly faTrash = faTrash;

  private readonly fb = inject(FormBuilder).nonNullable;

  public readonly fieldId = input.required<string>();
  public readonly label = input.required<string>();
  public readonly placeholder = input.required<string>();
  public readonly invalidHint = input.required<string>();
  public readonly removeAriaLabel = input.required<string>();
  public readonly domain = input<CustomDomain>();
  public readonly cnameTarget = input<string>();
  public readonly saving = input(false);

  public readonly save = output<string>();
  public readonly remove = output<void>();

  protected readonly control = this.fb.control('', [
    Validators.pattern(HOSTNAME_REGEX),
    Validators.maxLength(HOSTNAME_MAX_LENGTH),
  ]);

  protected submit() {
    this.control.markAsTouched();
    if (this.control.invalid || !this.control.value) {
      return;
    }
    this.save.emit(this.control.value.toLowerCase());
    this.control.reset();
  }
}
