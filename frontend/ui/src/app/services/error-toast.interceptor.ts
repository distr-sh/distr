import {HttpContext, HttpContextToken, HttpErrorResponse, HttpEventType, HttpInterceptorFn} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {captureException} from '@sentry/browser';
import {defer, EMPTY, finalize, Observable, retry, tap, timer} from 'rxjs';
import {displayedInToast} from '../../util/errors';
import {MaintenanceService} from './maintenance.service';
import {ToastRef, ToastService} from './toast.service';

class PollingHandle {
  onStop?: () => void;
}

const POLLING = new HttpContextToken<PollingHandle | undefined>(() => undefined);

/**
 * Runs a poll whose requests pass the given context. Their failures share a single toast that closes once every
 * failing poll has succeeded again or stopped, rather than adding a toast per failed attempt. A failure shown in
 * that toast is retried with backoff, since a poll that stopped on it could never recover.
 */
export function trackPolling<T>(poll: (context: HttpContext) => Observable<T>): Observable<T> {
  return defer(() => {
    const handle = new PollingHandle();
    return poll(new HttpContext().set(POLLING, handle)).pipe(
      retry({delay: (err, count) => (displayedInToast(err) ? timer(Math.min(count * count, 30) * 1000) : EMPTY)}),
      finalize(() => handle.onStop?.())
    );
  });
}

@Injectable({providedIn: 'root'})
class PollingFailureToast {
  private readonly toast = inject(ToastService);
  private readonly failing = new Set<PollingHandle>();
  // Kept after the user dismisses the toast, so that it stays closed until polling recovers.
  private ref?: ToastRef;
  private message?: string;
  private count = 0;

  failed(handle: PollingHandle, message: string) {
    this.failing.add(handle);
    handle.onStop = () => this.recovered(handle);
    this.count = message === this.message ? this.count + 1 : 1;
    this.message = message;
    const text = this.count > 1 ? `${message} (${this.count})` : message;
    if (this.ref) {
      this.ref.setMessage(text);
    } else {
      this.ref = this.toast.error(text);
    }
  }

  recovered(handle: PollingHandle) {
    if (!this.failing.delete(handle) || this.failing.size > 0) {
      return;
    }
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
  const pollingHandle = req.context.get(POLLING);
  return next(req).pipe(
    tap({
      next: (event) => {
        if (pollingHandle && event.type === HttpEventType.Response) {
          pollingFailureToast.recovered(pollingHandle);
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
        if (msg && pollingHandle) {
          pollingFailureToast.failed(pollingHandle, msg);
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
