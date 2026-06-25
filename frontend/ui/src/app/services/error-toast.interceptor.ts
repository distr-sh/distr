import {HttpErrorResponse, HttpInterceptorFn} from '@angular/common/http';
import {inject} from '@angular/core';
import {captureException} from '@sentry/browser';
import {tap} from 'rxjs';
import {displayedInToast} from '../../util/errors';
import {MaintenanceService} from './maintenance.service';
import {ToastService} from './toast.service';

export const errorToastInterceptor: HttpInterceptorFn = (req, next) => {
  const toast = inject(ToastService);
  const maintenance = inject(MaintenanceService);
  return next(req).pipe(
    tap({
      error: (err) => {
        // Ignore the readiness probe itself to avoid feedback loops.
        if (req.url.startsWith('/ready')) {
          return;
        }

        if (err.status && (err.status === 400 || err.status >= 500)) {
          captureException(err);
        }

        // A 5xx might mean the database is down. Let the maintenance service verify against the
        // readiness probe and either show the maintenance page or fall back to a toast.
        if (err instanceof HttpErrorResponse && err.status >= 500) {
          maintenance.handleServerError();
          return;
        }

        const msg = getToastDisplayedError(err);
        if (msg) {
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
