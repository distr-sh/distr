import {AbstractControl, FormGroup} from '@angular/forms';

export function enableControlsWithoutEvent(formGroup: FormGroup) {
  toggleControlsWithoutEvent(formGroup, true);
}

export function disableControlsWithoutEvent(formGroup: FormGroup) {
  toggleControlsWithoutEvent(formGroup, false);
}

export function toggleControlsWithoutEvent(formGroup: FormGroup, enabled: boolean) {
  if (enabled) {
    formGroup.enable({emitEvent: false});
  } else {
    formGroup.disable({emitEvent: false});
  }
}

/**
 * Returns true if the form is valid. If invalid, marks every control as touched
 * so error messages bound to `touched && invalid` become visible. Use as an
 * early-return guard at the top of submit handlers:
 *
 *   if (!validate(this.form)) return;
 */
export function validate(form: AbstractControl): boolean {
  if (form.invalid) {
    form.markAllAsTouched();
    return false;
  }
  return true;
}
