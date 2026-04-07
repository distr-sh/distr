import {DatePipe} from '@angular/common';
import {Component, computed, effect, inject, input, output, signal} from '@angular/core';
import {rxResource} from '@angular/core/rxjs-interop';
import {FormControl, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard, faClipboardCheck, faXmark} from '@fortawesome/free-solid-svg-icons';
import {EditorComponent} from '../../components/editor.component';
import {LicenseKeysService} from '../../services/license-keys.service';
import {ToastService} from '../../services/toast.service';
import {LicenseKey} from '../../types/license-key';

@Component({
  selector: 'app-view-license-key-modal',
  templateUrl: './view-license-key-modal.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, EditorComponent, DatePipe],
})
export class ViewLicenseKeyModalComponent {
  license = input.required<LicenseKey>();
  token = input.required<string>();
  closed = output<void>();

  activeTab = signal<'token' | 'payload' | 'decoded' | 'revisions'>('token');
  copied = false;

  private readonly licenseKeysService = inject(LicenseKeysService);

  revisionsResource = rxResource({
    params: () => ({licenseId: this.license().id!}),
    stream: ({params}) => this.licenseKeysService.getRevisions(params.licenseId),
  });

  protected readonly payloadControl = new FormControl({value: '', disabled: true});
  protected readonly decodedControl = new FormControl({value: '', disabled: true});
  protected readonly revisionPayloadControl = computed(() =>
    (this.revisionsResource.value() ?? [])?.map(
      (revision) => new FormControl({value: JSON.stringify(revision.payload, undefined, 2), disabled: true})
    )
  );

  protected readonly faXmark = faXmark;
  protected readonly faClipboard = faClipboard;
  protected readonly faClipboardCheck = faClipboardCheck;

  private readonly toast = inject(ToastService);

  constructor() {
    effect(() => {
      this.payloadControl.setValue(JSON.stringify(this.license().payload, null, 2));
      this.decodedControl.setValue(this.decodeToken(this.token()));
    });
  }

  private base64UrlDecode(input: string): string {
    let base64 = input.replace(/-/g, '+').replace(/_/g, '/');
    while (base64.length % 4 !== 0) {
      base64 += '=';
    }
    return atob(base64);
  }

  private decodeToken(token: string): string {
    try {
      const parts = token.split('.');
      if (parts.length < 2) {
        return token;
      }
      const header = JSON.parse(this.base64UrlDecode(parts[0]));
      const payload = JSON.parse(this.base64UrlDecode(parts[1]));
      return JSON.stringify({header, payload}, null, 2);
    } catch {
      return token;
    }
  }

  close() {
    this.closed.emit();
  }

  async copyToken() {
    try {
      await navigator.clipboard.writeText(this.token());
      this.toast.success('Copied to clipboard');
      this.copied = true;
      setTimeout(() => (this.copied = false), 2000);
    } catch {
      this.toast.error('Failed to copy to clipboard');
    }
  }
}
