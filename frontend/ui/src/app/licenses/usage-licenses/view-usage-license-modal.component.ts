import {Component, effect, inject, input, output, signal} from '@angular/core';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard, faClipboardCheck, faXmark} from '@fortawesome/free-solid-svg-icons';
import {EditorComponent} from '../../components/editor.component';
import {ToastService} from '../../services/toast.service';
import {UsageLicense} from '../../types/usage-license';

@Component({
  selector: 'app-view-usage-license-modal',
  templateUrl: './view-usage-license-modal.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, EditorComponent],
})
export class ViewUsageLicenseModalComponent {
  license = input.required<UsageLicense>();
  closed = output<void>();

  activeTab = signal<'token' | 'payload' | 'decoded'>('token');
  copied = false;

  payloadControl = new FormControl({value: '', disabled: true});
  decodedControl = new FormControl({value: '', disabled: true});

  protected readonly faXmark = faXmark;
  protected readonly faClipboard = faClipboard;
  protected readonly faClipboardCheck = faClipboardCheck;

  private readonly toast = inject(ToastService);

  constructor() {
    effect(() => {
      const license = this.license();
      this.payloadControl.setValue(JSON.stringify(license.payload, null, 2));
      this.decodedControl.setValue(this.decodeToken(license.token));
    });
  }

  private decodeToken(token: string): string {
    try {
      const parts = token.split('.');
      const header = JSON.parse(atob(parts[0]));
      const payload = JSON.parse(atob(parts[1]));
      return JSON.stringify({header, payload}, null, 2);
    } catch {
      return token;
    }
  }

  close() {
    this.closed.emit();
  }

  async copyToken() {
    await navigator.clipboard.writeText(this.license().token);
    this.toast.success('Copied to clipboard');
    this.copied = true;
    setTimeout(() => (this.copied = false), 2000);
  }
}
