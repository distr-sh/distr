import {never} from '../../util/exhaust';
import {BadgeVariant} from '../components/badge';
import {ConfirmConfig} from '../components/confirm-dialog/confirm-dialog.component';
import {SupportBundleStatus} from '../types/support-bundle';

export const supportBundleDeleteConfirm: ConfirmConfig = {
  message: {
    message: 'Are you sure you want to delete this support bundle?',
    alert: {type: 'warning', message: 'All collected resources and comments are deleted permanently.'},
  },
  confirmLabel: 'Delete',
};

export function supportBundleStatusVariant(status: SupportBundleStatus): BadgeVariant {
  switch (status) {
    case 'initialized':
      return 'info';
    case 'created':
      return 'warning';
    case 'resolved':
      return 'success';
    case 'canceled':
      return 'muted';
    default:
      return never(status);
  }
}
