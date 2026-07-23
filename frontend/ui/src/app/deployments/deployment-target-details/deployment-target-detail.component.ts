import {OverlayModule} from '@angular/cdk/overlay';
import {
  ChangeDetectionStrategy,
  Component,
  computed,
  effect,
  ElementRef,
  inject,
  signal,
  viewChild,
} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {AbstractControl, FormBuilder, ReactiveFormsModule, ValidationErrors} from '@angular/forms';
import {ActivatedRoute, Params, Router, RouterLink} from '@angular/router';
import {DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowDownWideShort,
  faArrowUpShortWide,
  faChevronDown,
  faDownload,
  faFilterCircleXmark,
  faPlay,
  faServer,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {combineLatest, debounceTime, map, of, switchMap, timer} from 'rxjs';
import {dateTimeLocalToISO, isoToDateTimeLocal} from '../../../util/dates';
import {AuthService} from '../../services/auth.service';
import {DeploymentLogsService} from '../../services/deployment-logs.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {OrganizationService} from '../../services/organization.service';
import {OrderDirection} from '../../types/timeseries-options';
import {DeploymentAppNameComponent} from '../deployment-target-card/deployment-app-name.component';
import {DeploymentLogsTableComponent} from './deployment-logs-table.component';
import {DeploymentStatusTableComponent} from './deployment-status-table.component';
import {DeploymentTargetLogsTableComponent} from './deployment-target-logs-table.component';

const ORDER_DIRECTION_KEY = 'logViewer.orderDirection';
const BUSINESS_LOG_BANNER_DISMISSED_KEY = 'logViewer.businessLogBannerDismissed';

@Component({
  selector: 'app-deployment-target-detail',
  templateUrl: './deployment-target-detail.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    DeploymentAppNameComponent,
    DeploymentLogsTableComponent,
    DeploymentStatusTableComponent,
    DeploymentTargetLogsTableComponent,
    FaIconComponent,
    OverlayModule,
    ReactiveFormsModule,
    RouterLink,
  ],
})
export class DeploymentTargetDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly deploymentLogsService = inject(DeploymentLogsService);
  private readonly organizationService = inject(OrganizationService);
  private readonly auth = inject(AuthService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faServer = faServer;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faDownload = faDownload;
  protected readonly faFilterCircleXmark = faFilterCircleXmark;
  protected readonly faPlay = faPlay;
  protected readonly faArrowDownWideShort = faArrowDownWideShort;
  protected readonly faArrowUpShortWide = faArrowUpShortWide;
  protected readonly faXmark = faXmark;
  protected readonly orderDirection = signal<OrderDirection>(
    (localStorage.getItem(ORDER_DIRECTION_KEY) as OrderDirection) || 'DESC'
  );
  protected readonly newestFirst = computed(() => this.orderDirection() === 'DESC');

  protected readonly targetDropdown = signal(false);
  protected targetDropdownWidth = 0;
  private readonly targetDropdownTrigger = viewChild.required<ElementRef<HTMLElement>>('targetDropdownTrigger');

  protected readonly deploymentDropdown = signal(false);
  protected deploymentDropdownWidth = 0;
  private readonly deploymentDropdownTrigger = viewChild.required<ElementRef<HTMLElement>>('deploymentDropdownTrigger');

  protected readonly resourceDropdown = signal(false);
  protected resourceDropdownWidth = 0;
  private readonly resourceDropdownTrigger = viewChild<ElementRef<HTMLElement>>('resourceDropdownTrigger');
  protected readonly showArchivedResources = signal(false);

  private readonly deploymentTargetId$ = this.route.paramMap.pipe(map((p) => p.get('deploymentTargetId')!));
  protected readonly deploymentTargetId = toSignal(this.deploymentTargetId$);
  private readonly deploymentId$ = this.route.queryParamMap.pipe(map((p) => p.get('deploymentId')));
  protected readonly deploymentId = toSignal(this.deploymentId$);
  private readonly selectedResources$ = this.route.queryParamMap.pipe(map((p) => p.getAll('resource')));
  protected readonly selectedResources = toSignal(this.selectedResources$, {initialValue: [] as string[]});
  private readonly fromParam = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('from') ?? undefined)));
  private readonly toParam = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('to') ?? undefined)));
  // Gate table inputs with the same window check so stale/bookmarked URLs never query out-of-window.
  protected readonly after = computed(() => this.validRangeDate(this.fromParam()));
  protected readonly before = computed(() => this.validRangeDate(this.toParam()));
  private readonly filter$ = this.route.queryParamMap.pipe(map((p) => p.get('filter') || undefined));
  protected readonly filter = toSignal(this.filter$);

  protected readonly live = computed(() => !this.after() && !this.before());
  // A present-but-invalid range must block the tables instead of falling back to live tailing.
  protected readonly rangeBlocked = computed(
    () => this.windowErrors(this.fromParam()) !== null || this.windowErrors(this.toParam()) !== null
  );

  private readonly organization = toSignal(this.organizationService.get());
  // Ticks every minute so time-based bounds/validation don't freeze in long sessions.
  private readonly now = toSignal(timer(0, 60_000).pipe(map(() => dayjs())), {initialValue: dayjs()});
  // Constrain the pickers to [start of the first day inside the window, now]; the range
  // always begins at 00:00 local time so users can select whole days. The backend allows
  // an extra day on top of the exact window to cover any timezone's midnight.
  protected readonly logRangeMin = computed(() => {
    const windowSeconds = this.organization()?.subscriptionLimits.logQueryWindowSeconds;
    return windowSeconds
      ? this.now().subtract(windowSeconds, 'second').startOf('day').format('YYYY-MM-DDTHH:mm')
      : '';
  });
  protected readonly logRangeMax = computed(() => this.now().format('YYYY-MM-DDTHH:mm'));

  private readonly businessLogBannerDismissed = signal(
    sessionStorage.getItem(BUSINESS_LOG_BANNER_DISMISSED_KEY) === 'true'
  );

  // Business plan upsell for vendor admins on plans with a shorter log window
  protected readonly showBusinessLogBanner = computed(() => {
    const subscriptionType = this.organization()?.subscriptionType;
    return (
      !this.businessLogBannerDismissed() &&
      this.auth.hasAnyRole('admin') &&
      this.auth.isVendor() &&
      (subscriptionType === 'pro' || subscriptionType === 'trial')
    );
  });

  protected dismissBusinessLogBanner(): void {
    sessionStorage.setItem(BUSINESS_LOG_BANNER_DISMISSED_KEY, 'true');
    this.businessLogBannerDismissed.set(true);
  }

  // Authoritative window check (the [min]/[max] attributes only guide the widget), shared by
  // the form validator and the table inputs.
  private readonly windowErrors = (value: string | undefined): ValidationErrors | null => {
    if (!value) {
      return null;
    }
    const date = dayjs(value);
    if (!date.isValid()) {
      return {invalidDate: true};
    }
    if (date.isAfter(this.now())) {
      return {afterNow: true};
    }
    const org = this.organization();
    if (!org) {
      // Org still loading: block sync/query until the window is known (no user-facing message).
      return {windowPending: true};
    }
    const windowSeconds = org.subscriptionLimits.logQueryWindowSeconds;
    if (windowSeconds && date.isBefore(this.now().subtract(windowSeconds, 'second').startOf('day'))) {
      return {beforeWindow: true};
    }
    return null;
  };

  private readonly logRangeValidator = (control: AbstractControl): ValidationErrors | null =>
    this.windowErrors(control.value as string);

  private validRangeDate(value: string | undefined): Date | undefined {
    if (!value || this.windowErrors(value)) {
      return undefined;
    }
    return new Date(value);
  }

  private readonly deploymentTargets$ = this.deploymentTargetsService.list();
  protected readonly deploymentTargets = toSignal(this.deploymentTargets$, {initialValue: []});
  protected readonly selectedDeploymentTarget = toSignal(
    combineLatest([this.deploymentTargets$, this.deploymentTargetId$]).pipe(
      map(([targets, id]) => targets.find((t) => t.id === id))
    )
  );

  protected readonly selectedDeployment = computed(() => {
    const id = this.deploymentId();
    return id ? this.selectedDeploymentTarget()?.deployments?.find((d) => d.id === id) : undefined;
  });

  protected readonly availableResources = toSignal(
    this.route.queryParamMap.pipe(
      map((p) => p.get('deploymentId')),
      switchMap((id) => (id ? this.deploymentLogsService.getResources(id) : of(null)))
    )
  );

  protected readonly visibleResources = computed(() => {
    const available = this.availableResources();
    if (!available) return [];
    const resources = [...(available.active ?? [])];
    if (this.showArchivedResources()) {
      resources.push(...(available.archived ?? []));
    }
    return resources;
  });

  protected readonly allVisibleResourcesSelected = computed(() => {
    const visible = this.visibleResources();
    if (visible.length === 0) return false;
    const selected = this.selectedResources();
    return visible.every((r) => selected.includes(r));
  });

  protected readonly someVisibleResourcesSelected = computed(() => {
    const visible = this.visibleResources();
    const selected = this.selectedResources();
    return visible.some((r) => selected.includes(r)) && !this.allVisibleResourcesSelected();
  });

  protected readonly form = this.fb.group({
    from: ['', this.logRangeValidator],
    to: ['', this.logRangeValidator],
    filter: '',
  });

  private readonly deploymentTargetLogsTable = viewChild(DeploymentTargetLogsTableComponent);
  private readonly deploymentStatusTable = viewChild(DeploymentStatusTableComponent);
  private readonly deploymentLogsTable = viewChild(DeploymentLogsTableComponent);

  constructor() {
    effect(() => localStorage.setItem(ORDER_DIRECTION_KEY, this.orderDirection()));

    // Validators can't read signals reactively, so re-validate when org or time changes.
    effect(() => {
      this.organization();
      this.now();
      this.form.controls.from.updateValueAndValidity({emitEvent: false});
      this.form.controls.to.updateValueAndValidity({emitEvent: false});
    });

    this.route.queryParamMap.pipe(takeUntilDestroyed()).subscribe((params) => {
      this.form.patchValue(
        {
          from: isoToDateTimeLocal(params.get('from')),
          to: isoToDateTimeLocal(params.get('to')),
          filter: params.get('filter') ?? '',
        },
        {emitEvent: false}
      );
    });

    this.form.valueChanges.pipe(takeUntilDestroyed(), debounceTime(300)).subscribe((values) => {
      const queryParams: Params = {filter: values.filter || null};
      // Only propagate valid dates so an invalid one doesn't block filter/other-date updates.
      if (this.form.controls.from.valid) {
        queryParams['from'] = dateTimeLocalToISO(values.from);
      }
      if (this.form.controls.to.valid) {
        queryParams['to'] = dateTimeLocalToISO(values.to);
      }
      this.router.navigate([], {
        relativeTo: this.route,
        queryParams,
        queryParamsHandling: 'merge',
      });
    });
  }

  protected toggleTargetDropdown() {
    this.targetDropdown.update((v) => !v);
    if (this.targetDropdown()) {
      this.targetDropdownWidth = this.targetDropdownTrigger().nativeElement.getBoundingClientRect().width;
    }
  }

  protected toggleDeploymentDropdown() {
    this.deploymentDropdown.update((v) => !v);
    if (this.deploymentDropdown()) {
      this.deploymentDropdownWidth = this.deploymentDropdownTrigger().nativeElement.getBoundingClientRect().width;
    }
  }

  protected toggleResourceDropdown() {
    this.resourceDropdown.update((v) => !v);
    if (this.resourceDropdown()) {
      const trigger = this.resourceDropdownTrigger();
      if (trigger) {
        this.resourceDropdownWidth = trigger.nativeElement.getBoundingClientRect().width;
      }
    }
  }

  protected selectDeployment(deployment: DeploymentWithLatestRevision | undefined) {
    this.form.patchValue({filter: ''});
    this.deploymentDropdown.set(false);
    this.resourceDropdown.set(false);
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {deploymentId: deployment?.id ?? null, resource: null},
      queryParamsHandling: 'merge',
    });
  }

  protected toggleResources(resources: string[]) {
    const current = this.selectedResources();
    const allSelected = resources.every((r) => current.includes(r));
    const updated = allSelected
      ? current.filter((r) => !resources.includes(r))
      : [...new Set([...current, ...resources])];
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {resource: updated.length > 0 ? updated : null},
      queryParamsHandling: 'merge',
    });
  }

  protected clearResources() {
    this.resourceDropdown.set(false);
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {resource: null},
      queryParamsHandling: 'merge',
    });
  }

  protected resetAllFilters() {
    this.form.reset();
  }

  protected resetDateFilters() {
    this.form.patchValue({from: '', to: ''});
  }

  protected export() {
    // Only one of the tables is shown at any given time, so it's fine to call export on all of them
    this.deploymentTargetLogsTable()?.export();
    this.deploymentStatusTable()?.export();
    this.deploymentLogsTable()?.export();
  }
}
