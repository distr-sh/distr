import {Component, forwardRef, input} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {ControlValueAccessor, FormControl, NG_VALUE_ACCESSOR, ReactiveFormsModule} from '@angular/forms';
import dayjs from 'dayjs';

interface ExpiresAtPreset {
  label: string;
  amount: number;
  unit: dayjs.ManipulateType;
}

const PRESETS: ExpiresAtPreset[] = [
  {label: '7 days', amount: 7, unit: 'day'},
  {label: '30 days', amount: 30, unit: 'day'},
  {label: '1 year', amount: 1, unit: 'year'},
];

@Component({
  selector: 'app-expires-at-picker',
  imports: [ReactiveFormsModule],
  templateUrl: './expires-at-picker.component.html',
  providers: [{provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => ExpiresAtPickerComponent), multi: true}],
})
export class ExpiresAtPickerComponent implements ControlValueAccessor {
  public readonly allowNoExpiration = input(true);
  public readonly id = input<string>();

  protected readonly presets = PRESETS;
  protected readonly control = new FormControl('', {nonNullable: true});

  constructor() {
    this.control.valueChanges.pipe(takeUntilDestroyed()).subscribe((value) => {
      this.onTouched();
      this.onChange(value);
    });
  }

  protected presetDate(preset: ExpiresAtPreset): string {
    return dayjs().add(preset.amount, preset.unit).startOf('day').format('YYYY-MM-DD');
  }

  writeValue(value: string | null | undefined): void {
    this.control.setValue(value ?? '', {emitEvent: false});
  }

  registerOnChange(fn: (value: string) => void): void {
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

  private onChange: (value: string) => void = () => {};
  private onTouched: () => void = () => {};
}
