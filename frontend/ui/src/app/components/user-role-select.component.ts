import {Component, computed, forwardRef, input} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {ControlValueAccessor, FormControl, NG_VALUE_ACCESSOR, ReactiveFormsModule} from '@angular/forms';
import {UserRole} from '@distr-sh/distr-sdk';
import {USER_ROLE_LABELS, userRolesAtOrBelow} from '../../util/user-role';

@Component({
  selector: 'app-user-role-select',
  imports: [ReactiveFormsModule],
  template: `
    <select [id]="id()" [attr.aria-label]="ariaLabel()" [class]="selectClass()" [formControl]="control">
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

  protected readonly control = new FormControl<UserRole | undefined>(undefined);
  protected readonly labels = USER_ROLE_LABELS;
  protected readonly options = computed<UserRole[]>(() => userRolesAtOrBelow(this.maxRole()));

  constructor() {
    this.control.valueChanges.pipe(takeUntilDestroyed()).subscribe((v) => {
      this.onTouched();
      this.onChange(v ?? undefined);
    });
  }

  writeValue(value: UserRole | undefined): void {
    this.control.setValue(value, {emitEvent: false});
  }

  registerOnChange(fn: (v: UserRole | undefined) => void): void {
    this.onChange = fn;
  }

  registerOnTouched(fn: () => void): void {
    this.onTouched = fn;
  }

  setDisabledState(isDisabled: boolean): void {
    if (isDisabled) {
      this.control.disable({emitEvent: false});
    } else {
      this.control.enable({emitEvent: false});
    }
  }

  private onChange: (v: UserRole | undefined) => void = () => {};
  private onTouched: () => void = () => {};
}
