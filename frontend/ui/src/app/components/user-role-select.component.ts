import {Component, computed, forwardRef, input, signal} from '@angular/core';
import {ControlValueAccessor, FormsModule, NG_VALUE_ACCESSOR} from '@angular/forms';
import {UserRole} from '@distr-sh/distr-sdk';
import {USER_ROLE_LABELS, userRolesAtOrBelow} from '../../util/user-role';

@Component({
  selector: 'app-user-role-select',
  imports: [FormsModule],
  template: `
    <select
      [id]="id()"
      [attr.aria-label]="ariaLabel()"
      [class]="selectClass()"
      [disabled]="isDisabled()"
      [ngModel]="value()"
      (ngModelChange)="setValue($event)"
      (blur)="onTouched()">
      @if (emptyOptionLabel(); as label) {
        <option [ngValue]="undefined">{{ label }}</option>
      }
      @for (role of options(); track role) {
        <option [ngValue]="role">{{ labels[role] }}</option>
      }
    </select>
  `,
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => UserRoleSelectComponent), multi: true}],
})
export class UserRoleSelectComponent implements ControlValueAccessor {
  public readonly maxRole = input<UserRole>();
  public readonly selectClass = input<string>('');
  public readonly id = input<string>();
  public readonly ariaLabel = input<string>();
  public readonly emptyOptionLabel = input<string>();

  protected readonly value = signal<UserRole | undefined>(undefined);
  protected readonly isDisabled = signal(false);
  protected readonly labels = USER_ROLE_LABELS;

  protected readonly options = computed<UserRole[]>(() => userRolesAtOrBelow(this.maxRole()));

  writeValue(value: UserRole | undefined): void {
    this.value.set(value);
  }

  registerOnChange(fn: (v: UserRole | undefined) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    this.isDisabled.set(isDisabled);
  }

  protected setValue(v: UserRole | undefined): void {
    this.value.set(v);
    this.onChange(v);
  }

  protected onChange: (v: UserRole | undefined) => void = () => {};
  protected onTouched: () => void = () => {};
}
