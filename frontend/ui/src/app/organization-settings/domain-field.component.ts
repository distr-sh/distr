import {ChangeDetectionStrategy, Component, computed, effect, inject, input, output, signal} from '@angular/core';
import {FormBuilder, FormsModule, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faRotate, faTrash, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {HOSTNAME_MAX_LENGTH, HOSTNAME_REGEX} from '../../util/validation';
import {ClipComponent} from '../components/clip.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {CustomDomain, CustomDomainVerification} from '../types/custom-domain';

// A single CNAME-configurable domain field: shows the existing domain with a remove button once
// configured, or a validated hostname input beforehand. Purely presentational — the domain list and
// the actual create/delete calls belong to the parent, which knows the domain type and scope.
@Component({
  selector: 'app-domain-field',
  templateUrl: './domain-field.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [FaIconComponent, FormsModule, ReactiveFormsModule, AutotrimDirective, ClipComponent],
})
export class DomainFieldComponent {
  protected readonly faTrash = faTrash;
  protected readonly faRotate = faRotate;
  protected readonly faCircleCheck = faCircleCheck;
  protected readonly faTriangleExclamation = faTriangleExclamation;

  private readonly fb = inject(FormBuilder).nonNullable;

  public readonly fieldId = input.required<string>();
  public readonly label = input.required<string>();
  public readonly placeholder = input.required<string>();
  public readonly invalidHint = input.required<string>();
  public readonly removeAriaLabel = input.required<string>();
  public readonly domain = input<CustomDomain>();
  // Arrives after the domain: the parent requests it separately, since it is a live DNS lookup.
  public readonly verification = input<CustomDomainVerification>();
  public readonly cnameTarget = input<string>();
  public readonly saving = input(false);
  public readonly verifying = input(false);

  // When set, the field stays collapsed behind a checkbox with this label until it is ticked. A domain
  // that already exists is always shown, checkbox or not.
  public readonly optionalCheckboxLabel = input<string>();

  public readonly save = output<string>();
  public readonly remove = output<void>();
  public readonly recheck = output<void>();

  protected readonly checkboxId = computed(() => `${this.fieldId()}Enabled`);
  protected readonly enabled = signal(false);
  protected readonly expanded = computed(() => !this.optionalCheckboxLabel() || !!this.domain() || this.enabled());

  protected checkedAtLabel(dnsCheckedAt: string): string {
    return dayjs(dnsCheckedAt).fromNow();
  }

  protected readonly control = this.fb.control('', [
    Validators.required,
    Validators.pattern(HOSTNAME_REGEX),
    Validators.maxLength(HOSTNAME_MAX_LENGTH),
  ]);

  // Not control.invalid: an empty field only disables the save button, and telling someone who has not
  // typed anything that their hostname is invalid would be wrong.
  protected malformed(): boolean {
    return this.control.hasError('pattern') || this.control.hasError('maxlength');
  }

  constructor() {
    // Clear only once saved; the parent's async save may fail and the typed value must survive that.
    effect(() => {
      if (this.domain()) {
        this.control.reset('', {emitEvent: false});
      }
    });
  }

  protected submit(event: Event) {
    // Not (ngSubmit): that would attach a template-driven NgForm to a form whose only control is
    // reactive, so the native submit is intercepted here instead.
    event.preventDefault();
    this.control.markAsTouched();
    if (this.control.invalid) {
      return;
    }
    this.save.emit(this.control.value.toLowerCase());
  }
}
