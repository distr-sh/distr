import {CustomerHomeActivityType} from './customer-home';

export interface ActivityFeedEntry {
  id: string;
  createdAt: string;
  type: CustomerHomeActivityType;
  resourceId: string;
  subject: string;
  detail?: string;
  userName?: string;
  userImageUrl?: string;
}
