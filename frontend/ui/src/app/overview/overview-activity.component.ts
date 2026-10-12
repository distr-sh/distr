import {Component, computed, input, signal} from '@angular/core';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {
  faBox,
  faChevronDown,
  faChevronUp,
  faClockRotateLeft,
  faCube,
  faDashboard,
  faDownload,
  faKey,
  faLifeRing,
} from '@fortawesome/free-solid-svg-icons';
import {never} from '../../util/exhaust';
import {
  ActivityTimelineComponent,
  ActivityTimelineEntry,
} from '../components/activity-timeline/activity-timeline.component';
import {OverviewActivity} from '../types/overview';

@Component({
  selector: 'app-overview-activity',
  imports: [ActivityTimelineComponent, FaIconComponent],
  templateUrl: './overview-activity.component.html',
})
export class OverviewActivityComponent {
  public readonly activity = input.required<OverviewActivity[]>();
  public readonly vendorName = input<string>();
  public readonly vendorLogo = input<string>();

  protected readonly faChevronDown = faChevronDown;
  protected readonly faChevronUp = faChevronUp;
  protected readonly faClockRotateLeft = faClockRotateLeft;

  protected readonly collapsedEntryCount = 5;
  protected readonly expanded = signal(false);

  protected readonly entries = computed<ActivityTimelineEntry[]>(() =>
    this.activity().map((activity) => {
      const byVendor =
        activity.type === 'application_version_created' ||
        activity.type === 'artifact_version_created' ||
        activity.type === 'license_key_created' ||
        activity.type === 'license_key_updated';
      return {
        id: `${activity.type}-${activity.id}`,
        createdAt: activity.createdAt,
        userName: byVendor ? this.vendorName() : activity.user?.name,
        userImageUrl: byVendor ? this.vendorLogo() : activity.user?.imageUrl,
        userImageIsLogo: byVendor,
        action: action(activity),
        icon: icon(activity),
        ...link(activity),
      };
    })
  );
  protected readonly visibleEntries = computed(() =>
    this.expanded() ? this.entries() : this.entries().slice(0, this.collapsedEntryCount)
  );
}

function icon({type}: OverviewActivity): IconDefinition {
  switch (type) {
    case 'application_version_created':
      return faCube;
    case 'artifact_version_created':
      return faBox;
    case 'artifact_version_first_pulled':
      return faDownload;
    case 'deployment_revision_created':
      return faDashboard;
    case 'license_key_created':
    case 'license_key_updated':
      return faKey;
    case 'support_bundle_created':
    case 'support_bundle_status_changed':
      return faLifeRing;
    default:
      return never(type);
  }
}

function link({type, resourceId, subject}: OverviewActivity): Pick<ActivityTimelineEntry, 'link' | 'queryParams'> {
  switch (type) {
    case 'application_version_created':
      return {link: '/deployments', queryParams: {search: subject}};
    case 'artifact_version_created':
    case 'artifact_version_first_pulled':
      return {link: `/artifacts/${resourceId}`};
    case 'deployment_revision_created':
      return {link: `/deployments/${resourceId}`};
    case 'license_key_created':
    case 'license_key_updated':
      return {link: '/license-keys'};
    case 'support_bundle_created':
    case 'support_bundle_status_changed':
      return {link: `/support/${resourceId}`};
    default:
      return never(type);
  }
}

function action({type, subject, detail}: OverviewActivity): string {
  switch (type) {
    case 'application_version_created':
      return `released version ${detail} of ${subject}`;
    case 'artifact_version_created':
      return `published ${subject}:${detail}`;
    case 'artifact_version_first_pulled':
      return `pulled ${subject}:${detail} for the first time`;
    case 'deployment_revision_created':
      return `deployed ${subject} ${detail}`;
    case 'license_key_created':
      return `issued the license key ${subject}`;
    case 'license_key_updated':
      return `updated the license key ${subject}`;
    case 'support_bundle_created':
      return `created the support bundle ${subject}`;
    case 'support_bundle_status_changed':
      switch (detail) {
        case 'resolved':
          return `resolved the support bundle ${subject}`;
        case 'canceled':
          return `canceled the support bundle ${subject}`;
        case 'created':
          return `uploaded the support bundle ${subject}`;
        default:
          return `changed the status of the support bundle ${subject} to ${detail}`;
      }
    default:
      return never(type);
  }
}
