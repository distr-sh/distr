// Customer Portal Compliance Hub — the primary home page for customer users.
// Expects data from: DeploymentTargetsService, ApplicationsService, OrganizationBrandingService, ContextService.
// Currently uses mock data for activity feed and some resource displays.
import {HttpErrorResponse} from '@angular/common/http';
import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowRight,
  faArrowUp,
  faBox,
  faBoxOpen,
  faCheck,
  faCircle,
  faCircleExclamation,
  faClockRotateLeft,
  faCubes,
  faDownload,
  faExclamationTriangle,
  faExternalLinkAlt,
  faFileLines,
  faFileShield,
  faRocket,
  faServer,
  faShieldHalved,
  faSpinner,
  faTag,
} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {catchError, EMPTY, map} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {isStale} from '../../../util/model';
import {AuthService} from '../../services/auth.service';
import {ContextService} from '../../services/context.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {ToastService} from '../../services/toast.service';
import {
  MOCK_ACTIVITY_FEED,
  MOCK_ARTIFACTS,
  MOCK_DEPLOYMENT_STATE,
  MOCK_DEPLOYMENTS,
  MOCK_RESOURCES,
  MockActivityEntry,
  MockArtifact,
  MockCVEReport,
  MockDeployment,
  MockResource,
} from './mock-data';

@Component({
  selector: 'app-home',
  imports: [FaIconComponent, RouterLink],
  templateUrl: './home.component.html',
})
export class HomeComponent {
  private readonly organizationBranding = inject(OrganizationBrandingService);
  private readonly toast = inject(ToastService);
  private readonly deploymentTargets = inject(DeploymentTargetsService);
  private readonly contextService = inject(ContextService);
  protected readonly auth = inject(AuthService);

  protected readonly faArrowUp = faArrowUp;
  protected readonly faArrowRight = faArrowRight;
  protected readonly faBox = faBox;
  protected readonly faBoxOpen = faBoxOpen;
  protected readonly faCheck = faCheck;
  protected readonly faCircle = faCircle;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faClockRotateLeft = faClockRotateLeft;
  protected readonly faCubes = faCubes;
  protected readonly faDownload = faDownload;
  protected readonly faExclamationTriangle = faExclamationTriangle;
  protected readonly faExternalLinkAlt = faExternalLinkAlt;
  protected readonly faFileLines = faFileLines;
  protected readonly faFileShield = faFileShield;
  protected readonly faRocket = faRocket;
  protected readonly faServer = faServer;
  protected readonly faShieldHalved = faShieldHalved;
  protected readonly faSpinner = faSpinner;
  protected readonly faTag = faTag;

  protected readonly brandingTitle = toSignal(
    this.organizationBranding.get().pipe(
      catchError((e) => {
        const msg = getFormDisplayedError(e);
        if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
          this.toast.error(msg);
        }
        return EMPTY;
      }),
      map((b) => b.title)
    )
  );

  protected readonly customerOrg = toSignal(this.contextService.getCustomerOrganization());

  protected readonly deploymentTargetList = toSignal(this.deploymentTargets.list(), {initialValue: []});

  // Computed: flatten all deployments across all targets for the status bar
  protected readonly allDeployments = computed(() => {
    return this.deploymentTargetList().flatMap((dt) =>
      dt.deployments.map((d) => ({
        ...d,
        targetName: dt.name,
        targetType: dt.type,
        targetStatus: dt.currentStatus,
        targetCreatedAt: dt.currentStatus?.createdAt,
      }))
    );
  });

  protected readonly firstDeployment = computed(() => {
    const deployments = this.allDeployments();
    return deployments.length > 0 ? deployments[0] : undefined;
  });

  protected readonly firstTarget = computed(() => {
    const targets = this.deploymentTargetList();
    return targets.length > 0 ? targets[0] : undefined;
  });

  protected readonly agentStatus = computed<'live' | 'stale' | 'disconnected'>(() => {
    const target = this.firstTarget();
    if (!target?.currentStatus) return 'disconnected';
    return isStale(target.currentStatus) ? 'stale' : 'live';
  });

  protected readonly agentHeartbeatAgo = computed(() => {
    const target = this.firstTarget();
    if (!target?.currentStatus?.createdAt) return '';
    return dayjs(target.currentStatus.createdAt).fromNow();
  });

  protected readonly hasDeployments = computed(() => this.allDeployments().length > 0);

  // MOCK DATA — replace with real API call for activity feed
  protected readonly mockState = MOCK_DEPLOYMENT_STATE;
  protected readonly activityFeed = signal<MockActivityEntry[]>(MOCK_ACTIVITY_FEED);
  protected readonly currentVersionResources = signal<MockResource[]>(
    MOCK_RESOURCES.filter((r) => r.version === MOCK_DEPLOYMENT_STATE.currentVersion)
  );

  protected readonly hasSbomResources = computed(() =>
    this.currentVersionResources().some((r) => r.type === 'sbom-spdx' || r.type === 'sbom-cyclonedx')
  );

  protected readonly sbomResources = computed(() =>
    this.currentVersionResources().filter((r) => r.type === 'sbom-spdx' || r.type === 'sbom-cyclonedx')
  );

  // MOCK DATA — CVE report for current version
  protected readonly currentVersionCVEReport = computed<MockCVEReport | undefined>(() => {
    const cveResource = this.currentVersionResources().find((r) => r.type === 'cve-report');
    return cveResource?.cveData;
  });

  protected readonly hasCVEReport = computed(() => this.currentVersionCVEReport() !== undefined);

  // MOCK DATA — deployments and artifacts
  protected readonly mockDeployments = signal<MockDeployment[]>(MOCK_DEPLOYMENTS);
  protected readonly mockArtifacts = signal<MockArtifact[]>(MOCK_ARTIFACTS);

  protected relativeTime(date: string): string {
    return dayjs(date).fromNow();
  }

  protected formatDate(date: string): string {
    return dayjs(date).format('MMM D, YYYY [at] h:mm A');
  }

  protected getResourceIcon(resource: MockResource) {
    switch (resource.type) {
      case 'release-notes':
        return this.faFileLines;
      case 'helm-chart':
        return this.faCubes;
      case 'sbom-spdx':
      case 'sbom-cyclonedx':
        return this.faFileShield;
      case 'guide':
        return this.faFileLines;
      case 'document':
        return this.faFileLines;
      case 'cve-report':
        return this.faFileShield;
    }
  }

  protected getResourceTypeLabel(resource: MockResource): string {
    switch (resource.type) {
      case 'release-notes':
        return 'Release Notes';
      case 'helm-chart':
        return 'Helm Chart';
      case 'sbom-spdx':
        return 'SBOM (SPDX)';
      case 'sbom-cyclonedx':
        return 'SBOM (CycloneDX)';
      case 'guide':
        return 'Guide';
      case 'document':
        return 'Document';
      case 'cve-report':
        return 'CVE Report';
    }
  }

  protected getActivityIcon(entry: MockActivityEntry) {
    switch (entry.type) {
      case 'deployment':
        return this.faRocket;
      case 'resource':
        return this.faFileLines;
      case 'release':
        return this.faTag;
    }
  }

  protected getActivityColor(entry: MockActivityEntry): string {
    switch (entry.type) {
      case 'deployment':
        return 'text-primary-600 dark:text-primary-400';
      case 'resource':
        return 'text-gray-500 dark:text-gray-400';
      case 'release':
        return 'text-green-600 dark:text-green-400';
    }
  }
}
