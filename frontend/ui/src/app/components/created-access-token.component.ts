import {Component, input, output} from '@angular/core';
import {AlertComponent} from './alert/alert.component';
import {ClipComponent} from './clip.component';

@Component({
  selector: 'app-created-access-token',
  imports: [ClipComponent, AlertComponent],
  template: `
    <app-alert type="success" [dismissible]="dismissible()" (dismissed)="dismissed.emit()">
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
    </app-alert>
  `,
})
export class CreatedAccessTokenComponent {
  public readonly tokenKey = input.required<string>();
  public readonly dismissible = input(false);
  public readonly dismissed = output<void>();
}
