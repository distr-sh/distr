import {HttpContext, HttpContextToken, HttpErrorResponse, HttpEventType, HttpInterceptorFn} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {captureException} from '@sentry/browser';
import {tap} from 'rxjs';
import {displayedInToast} from '../../util/errors';
import {MaintenanceService} from './maintenance.service';
import {ToastRef, ToastService} from './toast.service';

const POLLING = new HttpContextToken<boolean>(() => false);

// polling marks a request that repeats on a timer. Its failures share a single toast that closes once a
// polling request succeeds again, rather than adding a toast per failed attempt.
export function polling(): HttpContext {
  return new HttpContext().set(POLLING, true);
}

@Injectable({providedIn: 'root'})
class PollingFailureToast {
  private readonly toast = inject(ToastService);
  // Kept after the user dismisses the toast, so that it stays closed until polling recovers.
  private ref?: ToastRef;
  private message?: string;
  private count = 0;

  failed(message: string) {
    this.count = message === this.message ? this.count + 1 : 1;
    this.message = message;
    const text = this.count > 1 ? `${message} (${this.count})` : message;
    if (this.ref) {
      this.ref.setMessage(text);
    } else {
      this.ref = this.toast.error(text);
    }
  }

  recovered() {
    this.ref?.close();
    this.ref = undefined;
    this.message = undefined;
    this.count = 0;
  }
}

export const errorToastInterceptor: HttpInterceptorFn = (req, next) => {
  const toast = inject(ToastService);
  const pollingFailureToast = inject(PollingFailureToast);
  const maintenance = inject(MaintenanceService);
  const isPolling = req.context.get(POLLING);
  return next(req).pipe(
    tap({
      next: (event) => {
        if (isPolling && event.type === HttpEventType.Response) {
          pollingFailureToast.recovered();
        }
      },
      error: (err) => {
        // Ignore the readiness probe itself to avoid feedback loops.
        if (req.url.startsWith('/ready')) {
          return;
        }

        if (err.status && (err.status === 400 || err.status >= 500)) {
          captureException(err);
        }

        // A 5xx might mean the database is down: probe readiness (deduplicated) and show the maintenance
        // page if so. This runs independently of the toast below so every failure still gets feedback.
        if (err instanceof HttpErrorResponse && err.status >= 500) {
          maintenance.handleServerError();
        }

        const msg = getToastDisplayedError(err);
        if (msg && isPolling) {
          pollingFailureToast.failed(msg);
        } else if (msg) {
          toast.error(msg);
        }
      },
    })
  );
};

function getToastDisplayedError(err: any): string | undefined {
  if (displayedInToast(err) && err instanceof HttpErrorResponse) {
    switch (err.status) {
      case 429:
        const retryAfter = parseInt(err.headers.get('Retry-After') ?? '');
        if (!Number.isNaN(retryAfter)) {
          const minutes = Math.ceil(retryAfter / 60);
          return `Rate limited! Please try again in ${minutes} minute${minutes !== 1 ? 's' : ''}.`;
        }
        return 'Rate limited! Please try again later.';
      case 0:
        return 'Connection failed';
      default:
        return 'An unexpected technical error occurred';
    }
  }
  return;
}
