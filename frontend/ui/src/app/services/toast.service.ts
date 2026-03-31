import {inject, Injectable} from '@angular/core';
import {IndividualConfig, ToastrService} from 'ngx-toastr';
import {ToastComponent, ToastType} from '../components/toast.component';

const toastBaseConfig: Partial<IndividualConfig> = {
  toastComponent: ToastComponent,
  disableTimeOut: true,
  tapToDismiss: false,
  titleClass: '',
  messageClass: '',
  toastClass: '',
  positionClass: 'toast-bottom-right',
  easeTime: 150,
};

@Injectable({providedIn: 'root'})
export class ToastService {
  private readonly toastr = inject(ToastrService);

  public success(message: string) {
    this.toastr.show<ToastComponent, ToastType>('', message, {
      ...toastBaseConfig,
      payload: 'success',
      disableTimeOut: 'extendedTimeOut',
    });
  }

  public error(message: string) {
    this.toastr.show<ToastComponent, ToastType>('', message, {
      ...toastBaseConfig,
      payload: 'error',
    });
  }

  public info(message: string) {
    return this.toastr.show<ToastComponent, ToastType>('', message, {
      ...toastBaseConfig,
      payload: 'info',
    });
  }
}
