import {FormGroup} from '@angular/forms';

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
