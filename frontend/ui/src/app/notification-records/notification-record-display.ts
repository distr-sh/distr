import {NotificationRecord} from '../types/notification-record';

/**
 * notificationCustomerNames lists who a record concerns: the customer of an alert or of a customer's update
 * notification, otherwise every distinct customer whose deployments are behind.
 */
export function notificationCustomerNames(record: NotificationRecord): string[] {
  if (record.customerOrganizationName) {
    return [record.customerOrganizationName];
  }
  return [...new Set((record.deployments ?? []).map((d) => d.customerOrganizationName ?? 'Internal'))];
}
