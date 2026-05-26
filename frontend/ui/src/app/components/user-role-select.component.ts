import {Component, computed, forwardRef, input, signal} from '@angular/core';
import {ControlValueAccessor, FormsModule, NG_VALUE_ACCESSOR} from '@angular/forms';
import {UserRole} from '@distr-sh/distr-sdk';

const userRoleRank: Record<UserRole, number> = {
  read_only: 0,
  read_write: 1,
  admin: 2,
};

export const ALL_USER_ROLES: UserRole[] = ['read_only', 'read_write', 'admin'];

export const USER_ROLE_LABELS: Record<UserRole, string> = {
  read_only: 'Viewer',
  read_write: 'User',
  admin: 'Administrator',
};

@Component({
  selector: 'app-user-role-select',
  imports: [FormsModule],
  template: `
    <select
      [id]="id()"
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
  public readonly emptyOptionLabel = input<string>();

  protected readonly value = signal<UserRole | undefined>(undefined);
  protected readonly isDisabled = signal(false);
  protected readonly labels = USER_ROLE_LABELS;

  protected readonly options = computed<UserRole[]>(() => {
    const max = this.maxRole();
    if (!max) {
      return ALL_USER_ROLES;
    }
    const cap = userRoleRank[max];
    return ALL_USER_ROLES.filter((r) => userRoleRank[r] <= cap);
  });

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
