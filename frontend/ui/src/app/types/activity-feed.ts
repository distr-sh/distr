export type ActivityFeedEntryType =
  | 'application_version_created'
  | 'artifact_version_created'
  | 'artifact_version_first_pulled'
  | 'deployment_revision_created'
  | 'license_key_created'
  | 'license_key_updated'
  | 'support_bundle_created'
  | 'support_bundle_uploaded'
  | 'support_bundle_resolved'
  | 'support_bundle_canceled';

export interface ActivityFeedEntry {
  id: string;
  createdAt: string;
  type: ActivityFeedEntryType;
  resourceId: string;
  subject: string;
  detail?: string;
  userName?: string;
  userImageUrl?: string;
}
