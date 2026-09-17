import {Component, inject, input} from '@angular/core';
import {FormControl, ReactiveFormsModule, Validators} from '@angular/forms';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {PortalService} from '../services/portal.service';

export const mfaCodeValidators = [
  Validators.required,
  Validators.pattern(/^(\d{6}|\w{5}-\w{5})$/),
  Validators.minLength(6),
  Validators.maxLength(11),
];

export function newMfaCodeControl(): FormControl<string> {
  return new FormControl('', {nonNullable: true, validators: mfaCodeValidators});
}

@Component({
  selector: 'app-mfa-code-input',
  imports: [ReactiveFormsModule, AutotrimDirective],
  templateUrl: './mfa-code-input.component.html',
})
export class MfaCodeInputComponent {
  public readonly control = input.required<FormControl<string>>();
  protected readonly supportEmail = inject(PortalService).supportEmail;
}
