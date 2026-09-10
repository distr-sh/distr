import {Component, input, output} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import {ClipComponent} from './clip.component';

@Component({
  selector: 'app-created-access-token',
  imports: [ClipComponent, FaIconComponent],
  template: `
    <div
      class="flex items-start gap-4 p-4 text-sm text-green-800 rounded-lg bg-green-50 dark:bg-transparent dark:text-green-400 dark:border dark:border-green-400"
      role="alert">
      <div class="grow">
        <p>
          Your Personal Access Token:
          <code class="select-all" data-ph-mask-text="true">{{ tokenKey() }}</code>
          <app-clip class="mx-2" [clip]="tokenKey()" />
        </p>
        <p>
          <strong>Important:</strong>
          This is the only time you will be able to see this token, so please make sure to note it down before closing
          this page.
        </p>
      </div>
      @if (dismissible()) {
        <button
          type="button"
          aria-label="Dismiss"
          (click)="dismissed.emit()"
          class="-m-1 p-1.5 rounded-lg inline-flex items-center text-green-800 hover:bg-green-100 dark:text-green-400 dark:hover:bg-green-900">
          <fa-icon [icon]="faXmark" class="w-4 h-4" />
        </button>
      }
    </div>
  `,
})
export class CreatedAccessTokenComponent {
  protected readonly faXmark = faXmark;

  public readonly tokenKey = input.required<string>();
  public readonly dismissible = input(false);
  public readonly dismissed = output<void>();
}
