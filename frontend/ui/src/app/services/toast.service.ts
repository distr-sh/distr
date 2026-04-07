import {Injectable, signal} from '@angular/core';
import {ToastType} from '../components/toast.component';

export interface ToastEntry {
  id: string;
  type: ToastType;
  message: string;
  autoClose: boolean;
}

export interface ToastRef {
  close(): void;
}

@Injectable({providedIn: 'root'})
export class ToastService {
  readonly toasts = signal<ToastEntry[]>([]);
  private toastIdSequence = 0;

  public success(message: string): ToastRef {
    return this.add('success', message, true);
  }

  public error(message: string): ToastRef {
    return this.add('error', message, false);
  }

  public info(message: string): ToastRef {
    return this.add('info', message, false);
  }

  dismiss(id: string) {
    this.toasts.update((toasts) => toasts.filter((t) => t.id !== id));
  }

  private add(type: ToastType, message: string, autoClose: boolean): ToastRef {
    const id = (this.toastIdSequence++).toFixed();
    this.toasts.update((toasts) => [...toasts, {id, type, message, autoClose}]);
    return {close: () => this.dismiss(id)};
  }
}
