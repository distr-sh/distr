import {Component, input} from '@angular/core';
import {ClipComponent} from './clip.component';

@Component({
  selector: 'app-created-access-token',
  imports: [ClipComponent],
  template: `
    <div
      class="p-4 text-sm text-green-800 rounded-lg bg-green-50 dark:bg-transparent dark:text-green-400 dark:border dark:border-green-400"
      role="alert">
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
  `,
})
export class CreatedAccessTokenComponent {
  public readonly tokenKey = input.required<string>();
}
