import {NgTemplateOutlet} from '@angular/common';
import {Component, inject, TemplateRef} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {OverlayData} from '../../services/overlay.service';
import {ClosableDialog} from './closable-dialog';

export interface Message {
  message: string;
}

export interface Alert extends Message {
  type: 'warning' | 'danger';
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
  imports: [FaIconComponent, NgTemplateOutlet, AutotrimDirective, ReactiveFormsModule],
  templateUrl: './confirm-dialog.component.html',
})
export class ConfirmDialogComponent extends ClosableDialog<boolean> {
  protected readonly faXmark = faXmark;
  protected readonly data = inject(OverlayData) as ConfirmConfig;
  protected readonly confirmInput = new FormControl<string>('');

  protected readonly alertClass = [
    'p-4',
    'text-sm',
    'rounded-lg',
    ...(this.data.message?.alert?.type === 'warning'
      ? ['text-yellow-800', 'dark:text-yellow-300', 'bg-yellow-50', 'dark:bg-gray-800']
      : ['text-red-800', 'dark:text-red-400', 'bg-red-50', 'dark:bg-gray-800']),
  ];
}
