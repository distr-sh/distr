import {ActivityFeedEntry, ActivityFeedEntryType} from '../../types/activity-feed';
import {Actor} from '../../types/actor';

/** Who looks at the activity, which decides where its entries link to and whether they name their customer. */
export type ActivityFeedViewer = 'customer' | 'vendor' | 'partner';

export interface ActivityFeedCustomer {
  id: string;
  name: string;
}

export interface ActivityFeedPartner {
  id: string;
  /** Missing for a partner looking at its own members. */
  name?: string;
}

export interface ActivityFeedItem {
  id: string;
  createdAt: string;
  type: ActivityFeedEntryType;
  resourceId: string;
  resourceName: string;
  subject: string;
  detail?: string;
  user?: Actor;
  customer?: ActivityFeedCustomer;
  partner?: ActivityFeedPartner;
}

export function toActivityFeedItems(entries: ActivityFeedEntry[]): ActivityFeedItem[] {
  return entries.map((entry) => ({
    id: entry.id,
    createdAt: entry.createdAt,
    type: entry.type,
    resourceId: entry.resourceId,
    resourceName: entry.resourceName,
    subject: entry.subject,
    detail: entry.detail,
    user: entry.user,
    customer:
      entry.customerOrganizationId && entry.customerOrganizationName
        ? {id: entry.customerOrganizationId, name: entry.customerOrganizationName}
        : undefined,
    partner: entry.partnerOrganizationId
      ? {id: entry.partnerOrganizationId, name: entry.partnerOrganizationName}
      : undefined,
  }));
}
