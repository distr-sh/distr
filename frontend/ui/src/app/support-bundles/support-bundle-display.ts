import {never} from '../../util/exhaust';
import {ConfirmConfig} from '../components/confirm-dialog/confirm-dialog.component';
import {SupportBundleStatus} from '../types/support-bundle';

export const supportBundleDeleteConfirm: ConfirmConfig = {
  message: {
    message: 'Are you sure you want to delete this support bundle?',
    alert: {type: 'danger', message: 'All collected resources and comments are deleted permanently.'},
  },
  confirmLabel: 'Delete',
};

export function supportBundleStatusBadgeClass(status: SupportBundleStatus): string {
  switch (status) {
    case 'initialized':
      return 'bg-blue-100 text-blue-800 border-blue-400 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800';
    case 'created':
      return 'bg-yellow-100 text-yellow-800 border-yellow-400 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800';
    case 'resolved':
      return 'bg-green-100 text-green-800 border-green-400 dark:bg-green-900 dark:text-green-300 dark:border-green-800';
    case 'canceled':
      return 'bg-gray-100 text-gray-800 border-gray-400 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-600';
    default:
      return never(status);
  }
}
