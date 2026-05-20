import {Component, input} from '@angular/core';
import {AbstractControl} from '@angular/forms';

@Component({
  selector: 'app-latin1-error',
  template: `
    @if (control().hasError('latin1')) {
      <p class="mt-1 text-sm text-red-600 dark:text-red-500">
        Only Latin-1 characters are supported. Please remove non-ASCII characters such as the emdash <code>—</code> or
        smart quotes.
      </p>
    }
  `,
})
export class Latin1ErrorComponent {
  public readonly control = input.required<AbstractControl>();
}
