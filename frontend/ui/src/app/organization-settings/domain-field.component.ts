import {ChangeDetectionStrategy, Component, effect, inject, input, output} from '@angular/core';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleCheck, faRotate, faTrash, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {HOSTNAME_MAX_LENGTH, HOSTNAME_REGEX} from '../../util/validation';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {CustomDomain, CustomDomainVerification} from '../types/custom-domain';

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

  public readonly save = output<string>();
  public readonly remove = output<void>();
  public readonly recheck = output<void>();

  protected checkedAtLabel(dnsCheckedAt: string): string {
    return dayjs(dnsCheckedAt).fromNow();
  }

  protected readonly control = this.fb.control('', [
    Validators.pattern(HOSTNAME_REGEX),
    Validators.maxLength(HOSTNAME_MAX_LENGTH),
  ]);

  constructor() {
    // Clear only once saved; the parent's async save may fail and the typed value must survive that.
    effect(() => {
      if (this.domain()) {
        this.control.reset('', {emitEvent: false});
      }
    });
  }

  protected submit(event: Event) {
    // ngSubmit needs FormsModule to intercept the native submit and prevent the page from
    // reloading; this form only uses ReactiveFormsModule, so preventDefault directly instead.
    event.preventDefault();
    this.control.markAsTouched();
    if (this.control.invalid || !this.control.value) {
      return;
    }
    this.save.emit(this.control.value.toLowerCase());
  }
}
