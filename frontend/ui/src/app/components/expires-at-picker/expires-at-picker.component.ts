import {ChangeDetectionStrategy, Component, ElementRef, forwardRef, input, signal, viewChild} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {
  ControlValueAccessor,
  FormControl,
  NG_VALIDATORS,
  NG_VALUE_ACCESSOR,
  ReactiveFormsModule,
  ValidationErrors,
  Validator,
} from '@angular/forms';
import dayjs from 'dayjs';

interface ExpiresAtPreset {
  label: string;
  amount: number;
  unit: dayjs.ManipulateType;
}

// The format of an <input type="date"> value, and therefore of this control's value.
export const EXPIRES_AT_DATE_FORMAT = 'YYYY-MM-DD';

const PRESETS: ExpiresAtPreset[] = [
  {label: '7 days', amount: 7, unit: 'day'},
  {label: '30 days', amount: 30, unit: 'day'},
  {label: '1 year', amount: 1, unit: 'year'},
];

@Component({
  selector: 'app-expires-at-picker',
  imports: [ReactiveFormsModule],
  templateUrl: './expires-at-picker.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  providers: [
    {provide: NG_VALUE_ACCESSOR, useExisting: forwardRef(() => ExpiresAtPickerComponent), multi: true},
    {provide: NG_VALIDATORS, useExisting: forwardRef(() => ExpiresAtPickerComponent), multi: true},
  ],
})
export class ExpiresAtPickerComponent implements ControlValueAccessor, Validator {
  public readonly allowNoExpiration = input(true);
  public readonly inputId = input<string>();

  // A date is read as the local midnight it starts at, so the earliest one that leaves any validity
  // at all is tomorrow.
  protected readonly minDate = dayjs().add(1, 'day').format(EXPIRES_AT_DATE_FORMAT);
  protected readonly presets = PRESETS.map((preset) => ({
    label: preset.label,
    date: dayjs().add(preset.amount, preset.unit).startOf('day').format(EXPIRES_AT_DATE_FORMAT),
  }));
  protected readonly control = new FormControl('', {nonNullable: true});
  protected readonly pastDate = signal(false);

  private readonly dateInput = viewChild<ElementRef<HTMLInputElement>>('dateInput');

  constructor() {
    this.control.valueChanges.pipe(takeUntilDestroyed()).subscribe((value) => {
      // A date input reports an entry it cannot parse yet as an empty value, so every keystroke on
      // the way to a complete date would otherwise read as "no expiration".
      if (value === '' && this.dateInput()?.nativeElement.validity.badInput) {
        return;
      }
      this.pastDate.set(this.isPastDate(value));
      this.onTouched();
      this.onChange(value);
    });
  }

  writeValue(value: string | null | undefined): void {
    // Only a date the user picked is rejected. One that is already stored is a fact, and reporting
    // it would keep everything else on the form from being saved.
    this.pastDate.set(false);
    this.control.setValue(value ?? '', {emitEvent: false});
  }

  validate(): ValidationErrors | null {
    return this.pastDate() ? {expiresAtInPast: true} : null;
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

  private isPastDate(value: string): boolean {
    return value !== '' && dayjs(value).isBefore(dayjs(this.minDate));
  }

  private onChange: (value: string) => void = () => {};
  private onTouched: () => void = () => {};
}
