import {Component, inject} from '@angular/core';
import {ToastService} from '../services/toast.service';
import {ToastComponent} from './toast.component';

@Component({
  selector: 'app-toast-container',
  template: `
    <div class="fixed bottom-4 right-4 z-1001 flex flex-col items-end pointer-events-none">
      @for (toast of toastService.toasts(); track toast.id) {
        <app-toast
          [id]="toast.id"
          [type]="toast.type"
          [message]="toast.message"
          [autoClose]="toast.autoClose"
          class="pointer-events-auto"
          animate.enter="animate-fly-skew-in-right"
          animate.leave="animate-fly-out-right" />
      }
    </div>
  `,
  imports: [ToastComponent],
})
export class ToastContainerComponent {
  protected readonly toastService = inject(ToastService);
}
