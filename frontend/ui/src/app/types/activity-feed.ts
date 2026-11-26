import {Actor} from './actor';

export type ActivityFeedEntryType =
  | 'application_version_created'
  | 'artifact_version_created'
  | 'artifact_version_first_pulled'
  | 'customer_created'
  | 'customer_user_added'
  | 'deployment_revision_created'
  | 'deployment_target_created'
  | 'license_key_created'
  | 'license_key_updated'
  | 'support_bundle_created'
  | 'support_bundle_uploaded'
  | 'support_bundle_resolved'
  | 'team_member_added';

export interface ActivityFeedEntry {
  id: string;
  createdAt: string;
  type: ActivityFeedEntryType;
  resourceId: string;
  resourceName: string;
  subject: string;
  detail?: string;
  user?: Actor;
  customerOrganizationId?: string;
  customerOrganizationName?: string;
  partnerOrganizationId?: string;
  partnerOrganizationName?: string;
}
