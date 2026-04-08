import {Component, inject, input} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard, faClipboardCheck} from '@fortawesome/free-solid-svg-icons';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-clip',
  imports: [FaIconComponent],
  template: `
    <button
      type="button"
      (click)="writeClip()"
      class="text-gray-500 hover:text-gray-400 dark:text-gray-400 hover:dark:text-gray-300"
      title="Copy to clipboard">
      @if (!copied) {
        <fa-icon [icon]="faClipboard" />
      } @else {
        <fa-icon [icon]="faClipboardCheck" />
      }
      <ng-content />
    </button>
  `,
})
export class ClipComponent {
  public readonly clip = input.required<string>();

  private readonly toast = inject(ToastService);

  protected readonly faClipboard = faClipboard;
  protected readonly faClipboardCheck = faClipboardCheck;

  protected copied = false;

  public async writeClip() {
    await navigator.clipboard.writeText(this.clip());
    this.toast.success('Copied to clipboard');
    this.copied = true;
    setTimeout(() => (this.copied = false), 2000);
  }
}
