import {inject} from '@angular/core';
import {ToastService} from '../app/services/toast.service';
import {getFormDisplayedError} from './errors';

/**
 * Lets a list page show its cached lists at once and replace them when the given refreshes arrive. Has to be called
 * in an injection context.
 */
export function refreshInBackground(...refreshes: Promise<unknown>[]): void {
  const toast = inject(ToastService);
  Promise.all(refreshes).catch((e) => {
    const msg = getFormDisplayedError(e);
    if (msg) {
      toast.error(msg);
    }
  });
}
