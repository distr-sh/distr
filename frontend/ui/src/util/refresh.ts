import {inject, WritableSignal} from '@angular/core';
import {DialogRef} from '../app/services/overlay.service';
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

/**
 * Opens an overlay with `loading` set and keeps it set while `load` runs. Resolves to `undefined` when the overlay was
 * closed in the meantime, so the caller does not fill a form nobody sees anymore.
 */
export async function loadWhileOpen<T, R>(
  loading: WritableSignal<boolean>,
  open: () => DialogRef<R>,
  load: Promise<T>
): Promise<T | undefined> {
  loading.set(true);
  let isOpen = true;
  const subscription = open()
    .result()
    .subscribe(() => (isOpen = false));
  try {
    const result = await load;
    return isOpen ? result : undefined;
  } finally {
    subscription.unsubscribe();
    if (isOpen) {
      loading.set(false);
    }
  }
}
