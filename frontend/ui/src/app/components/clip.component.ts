import {Component, Directive, inject, input, Signal, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard, faClipboardCheck} from '@fortawesome/free-solid-svg-icons';
import {ToastService} from '../services/toast.service';

abstract class ClipBase {
  public abstract readonly clip: Signal<string>;
  private readonly _copied = signal(false);
  public readonly copied = this._copied.asReadonly();
  private readonly toast = inject(ToastService);

  public async writeClip() {
    try {
      await navigator.clipboard.writeText(this.clip());
      this.toast.success('Copied to clipboard');
      this._copied.set(true);
      setTimeout(() => this._copied.set(false), 2000);
    } catch (e) {
      // Can happen e.g. if running locally (no HTTPS) with a different host than localhost
      this.toast.error('Failed to copy to clipboard');
    }
  }
}

@Directive({selector: '[appClip]', exportAs: 'clip'})
export class ClipDirective extends ClipBase {
  public override clip = input.required<string>({alias: 'appClip'});
}

@Component({
  selector: 'app-clip',
  imports: [FaIconComponent],
  template: `
    <button
      type="button"
      (click)="writeClip()"
      class="text-gray-500 hover:text-gray-400 dark:text-gray-400 hover:dark:text-gray-300"
      title="Copy to clipboard">
      <fa-icon [icon]="copied() ? faClipboardCheck : faClipboard" />
      <ng-content />
    </button>
  `,
})
export class ClipComponent extends ClipBase {
  public override clip = input.required<string>();
  protected readonly faClipboard = faClipboard;
  protected readonly faClipboardCheck = faClipboardCheck;
}
