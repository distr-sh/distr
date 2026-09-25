import {never} from '../../util/exhaust';
import {NotificationSourceType} from '../types/notification-record';

export function notificationKindLabel(sourceType: NotificationSourceType): string {
  switch (sourceType) {
    case 'alert':
      return 'Alert';
    case 'application':
    case 'artifact':
      return 'Update';
    default:
      return never(sourceType);
  }
}

export function notificationKindBadgeClass(sourceType: NotificationSourceType): string {
  return sourceType === 'alert'
    ? 'bg-orange-100 text-orange-800 border-orange-400 dark:bg-orange-900 dark:text-orange-300 dark:border-orange-800'
    : 'bg-blue-100 text-blue-800 border-blue-400 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800';
}
