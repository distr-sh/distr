import {never} from '../../util/exhaust';
import {BadgeVariant} from '../components/badge';
import {ConfirmConfig} from '../components/confirm-dialog/confirm-dialog.component';
import {SupportBundleStatus} from '../types/support-bundle';

export const supportBundleDeleteConfirm: ConfirmConfig = {
  message: {
    message: 'Are you sure you want to delete this support request?',
    alert: {type: 'warning', message: 'All collected resources and comments are deleted permanently.'},
  },
  confirmLabel: 'Delete',
};

export type SupportBundleProgress = 'awaiting_upload' | 'uploaded' | 'resolved';

export function supportBundleProgress(status: SupportBundleStatus, resourceCount: number): SupportBundleProgress {
  switch (status) {
    case 'open':
      return resourceCount > 0 ? 'uploaded' : 'awaiting_upload';
    case 'resolved':
      return 'resolved';
    default:
      return never(status);
  }
}

export function supportBundleProgressVariant(progress: SupportBundleProgress): BadgeVariant {
  switch (progress) {
    case 'awaiting_upload':
      return 'neutral';
    case 'uploaded':
      return 'info';
    case 'resolved':
      return 'success';
    default:
      return never(progress);
  }
}
