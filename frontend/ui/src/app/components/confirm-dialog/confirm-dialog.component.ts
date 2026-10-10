import {NgTemplateOutlet} from '@angular/common';
import {Component, computed, inject, TemplateRef} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {OverlayData} from '../../services/overlay.service';
import {AlertComponent, AlertType} from '../alert/alert.component';
import {ClosableDialog} from './closable-dialog';

export interface Message {
  message: string;
}

export interface Alert extends Message {
  type: AlertType;
}

export interface ConfirmMessage extends Message {
  alert?: Alert;
}

export interface ConfirmConfig {
  message?: ConfirmMessage;
  customTemplate?: TemplateRef<any>;
  requiredConfirmInputText?: string;
  confirmLabel?: string;
  cancelLabel?: string;
}

@Component({
  imports: [FaIconComponent, NgTemplateOutlet, AutotrimDirective, ReactiveFormsModule, AlertComponent],
  templateUrl: './confirm-dialog.component.html',
})
export class ConfirmDialogComponent extends ClosableDialog<boolean> {
  protected readonly faXmark = faXmark;
  protected readonly data = inject(OverlayData) as ConfirmConfig;
  protected readonly confirmInput = new FormControl<string>('', {nonNullable: true});

  // Both sides are compared trimmed, so a name that was stored with surrounding whitespace before
  // inputs were trimmed can still be confirmed.
  protected readonly requiredConfirmInputText = this.data.requiredConfirmInputText?.trim();
  private readonly confirmInputValue = toSignal(this.confirmInput.valueChanges, {initialValue: ''});
  protected readonly confirmDisabled = computed(
    () => !!this.requiredConfirmInputText && this.requiredConfirmInputText !== this.confirmInputValue().trim()
  );
}
