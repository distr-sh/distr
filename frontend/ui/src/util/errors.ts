import {HttpErrorResponse} from '@angular/common/http';
import {catchError, of, OperatorFunction} from 'rxjs';
import {ToastService} from '../app/services/toast.service';

export function displayedInToast(err: any): boolean {
  if (err instanceof HttpErrorResponse) {
    return !err.status || err.status === 429 || err.status >= 500;
  }
  return false;
}

export function isNotFound(err: unknown): boolean {
  return err instanceof HttpErrorResponse && err.status === 404;
}

export function getFormDisplayedError(err: any): string | undefined {
  if (!displayedInToast(err)) {
    if (err instanceof HttpErrorResponse && typeof err.error === 'string') {
      return err.error;
    } else {
      return 'Something went wrong';
    }
  }
  return;
}

export function toastDisplayedError(toast: ToastService, err: unknown) {
  const msg = getFormDisplayedError(err);
  if (msg) {
    toast.error(msg);
  }
}

export function recoverWithToast<T, F>(toast: ToastService, fallback: F): OperatorFunction<T, T | F> {
  return catchError((err) => {
    toastDisplayedError(toast, err);
    return of(fallback);
  });
}
