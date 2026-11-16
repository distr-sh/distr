import {ActivityFeedEntry, ActivityFeedEntryType} from '../types/activity-feed';

/** Who looks at the activity, which decides where its entries link to and whether they name their customer. */
export type HomeActivityViewer = 'customer' | 'vendor' | 'partner';

export interface HomeActivityUser {
  name: string;
  imageUrl?: string;
}

export interface HomeActivityCustomer {
  id: string;
  name: string;
}

export interface HomeActivity {
  id: string;
  createdAt: string;
  type: ActivityFeedEntryType;
  resourceId: string;
  subject: string;
  detail?: string;
  user?: HomeActivityUser;
  customer?: HomeActivityCustomer;
}

export function toHomeActivity(entries: ActivityFeedEntry[]): HomeActivity[] {
  return entries.map((entry) => ({
    id: entry.id,
    createdAt: entry.createdAt,
    type: entry.type,
    resourceId: entry.resourceId,
    subject: entry.subject,
    detail: entry.detail,
    user: entry.userName ? {name: entry.userName, imageUrl: entry.userImageUrl} : undefined,
    customer:
      entry.customerOrganizationId && entry.customerOrganizationName
        ? {id: entry.customerOrganizationId, name: entry.customerOrganizationName}
        : undefined,
  }));
}
