import {DatePipe} from '@angular/common';
import {Component, computed, input, signal} from '@angular/core';
import {Params, RouterLink} from '@angular/router';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {
  faBox,
  faClockRotateLeft,
  faCube,
  faDashboard,
  faDownload,
  faKey,
  faLifeRing,
} from '@fortawesome/free-solid-svg-icons';
import {RelativeDatePipe} from '../../../util/dates';
import {never} from '../../../util/exhaust';
import {AvatarComponent} from '../../components/avatar.component';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {CustomerHomeActivity} from './customer-home';
import {CustomerHomeCollapsibleComponent} from './customer-home-collapsible.component';

interface ActivityEntry {
  id: string;
  createdAt: string;
  action: string;
  icon: IconDefinition;
  link: string;
  queryParams?: Params;
  user?: {name: string; imageUrl?: string; isVendor: boolean};
}

@Component({
  selector: 'app-customer-home-activity',
  imports: [
    DatePipe,
    RouterLink,
    FaIconComponent,
    RelativeDatePipe,
    AvatarComponent,
    CardHeadingComponent,
    CustomerHomeCollapsibleComponent,
  ],
  templateUrl: './customer-home-activity.component.html',
  host: {
    class: 'block',
    '[class.lg:absolute]': 'fillHeight() && !expanded()',
    '[class.lg:inset-0]': 'fillHeight() && !expanded()',
  },
})
export class CustomerHomeActivityComponent {
  public readonly activity = input.required<CustomerHomeActivity[]>();
  public readonly vendorName = input<string>();
  public readonly vendorLogo = input<string>();
  /**
   * Collapsed, the card takes the height of its positioned parent instead of its own, which places it next to the
   * other cards at their height. The parent's grid row then has to be sized by those cards alone.
   */
  public readonly fillHeight = input(false);

  protected readonly faClockRotateLeft = faClockRotateLeft;

  protected readonly expanded = signal(false);

  protected readonly entries = computed<ActivityEntry[]>(() =>
    this.activity().map((activity) => {
      const byVendor =
        activity.type === 'application_version_created' ||
        activity.type === 'artifact_version_created' ||
        activity.type === 'license_key_created' ||
        activity.type === 'license_key_updated' ||
        activity.type === 'support_bundle_resolved';
      const vendorName = this.vendorName();
      return {
        id: `${activity.type}-${activity.id}`,
        createdAt: activity.createdAt,
        action: action(activity),
        icon: icon(activity),
        ...link(activity),
        user: byVendor
          ? vendorName
            ? {name: vendorName, imageUrl: this.vendorLogo(), isVendor: true}
            : undefined
          : activity.user && {...activity.user, isVendor: false},
      };
    })
  );
}

function icon({type}: CustomerHomeActivity): IconDefinition {
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
    case 'support_bundle_uploaded':
    case 'support_bundle_resolved':
    case 'support_bundle_canceled':
      return faLifeRing;
    default:
      return never(type);
  }
}

function link({type, resourceId, subject}: CustomerHomeActivity): Pick<ActivityEntry, 'link' | 'queryParams'> {
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
    case 'support_bundle_uploaded':
    case 'support_bundle_resolved':
    case 'support_bundle_canceled':
      return {link: `/support/${resourceId}`};
    default:
      return never(type);
  }
}

function action({type, subject, detail}: CustomerHomeActivity): string {
  switch (type) {
    case 'application_version_created':
      return `version ${detail} of ${subject} released`;
    case 'artifact_version_created':
      return `artifact ${subject}:${detail} published`;
    case 'artifact_version_first_pulled':
      return `artifact ${subject}:${detail} pulled for the first time`;
    case 'deployment_revision_created':
      return `version ${detail} of ${subject} deployed`;
    case 'license_key_created':
      return `license key ${subject} issued`;
    case 'license_key_updated':
      return `license key ${subject} updated`;
    case 'support_bundle_created':
      return `support bundle ${subject} created`;
    case 'support_bundle_uploaded':
      return `support bundle ${subject} uploaded`;
    case 'support_bundle_resolved':
      return `support bundle ${subject} resolved`;
    case 'support_bundle_canceled':
      return `support bundle ${subject} canceled`;
    default:
      return never(type);
  }
}
