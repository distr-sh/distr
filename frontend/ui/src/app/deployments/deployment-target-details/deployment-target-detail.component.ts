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
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
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
} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {combineLatest, debounceTime, map, of, switchMap} from 'rxjs';
import {dateTimeLocalToISO, isoToDateTimeLocal} from '../../../util/dates';
import {DeploymentLogsService} from '../../services/deployment-logs.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {OrganizationService} from '../../services/organization.service';
import {OrderDirection} from '../../types/timeseries-options';
import {DeploymentAppNameComponent} from '../deployment-target-card/deployment-app-name.component';
import {DeploymentLogsTableComponent} from './deployment-logs-table.component';
import {DeploymentStatusTableComponent} from './deployment-status-table.component';
import {DeploymentTargetLogsTableComponent} from './deployment-target-logs-table.component';

const ORDER_DIRECTION_KEY = 'logViewer.orderDirection';

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
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faServer = faServer;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faDownload = faDownload;
  protected readonly faFilterCircleXmark = faFilterCircleXmark;
  protected readonly faPlay = faPlay;
  protected readonly faArrowDownWideShort = faArrowDownWideShort;
  protected readonly faArrowUpShortWide = faArrowUpShortWide;
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
  private readonly after$ = this.route.queryParamMap.pipe(
    map((p) => (p.has('from') ? new Date(p.get('from')!) : undefined))
  );
  protected readonly after = toSignal(this.after$);
  private readonly before$ = this.route.queryParamMap.pipe(
    map((p) => (p.has('to') ? new Date(p.get('to')!) : undefined))
  );
  protected readonly before = toSignal(this.before$);
  private readonly filter$ = this.route.queryParamMap.pipe(map((p) => p.get('filter') || undefined));
  protected readonly filter = toSignal(this.filter$);

  protected readonly live = computed(() => !this.after() && !this.before());

  private readonly organization = toSignal(this.organizationService.get());
  // The log query window is subscription-bound and enforced server-side. Constrain the
  // date pickers to [now - window, now] so users cannot select an out-of-window range
  // that the backend would reject.
  protected readonly logRangeMin = computed(() => {
    const windowSeconds = this.organization()?.subscriptionLimits.logQueryWindowSeconds;
    return windowSeconds ? dayjs().subtract(windowSeconds, 'second').format('YYYY-MM-DDTHH:mm') : '';
  });
  protected readonly logRangeMax = computed(() => dayjs().format('YYYY-MM-DDTHH:mm'));

  // The [min]/[max] picker attributes only guide the native widget; they do not stop a
  // typed, pasted or bookmarked out-of-window value. This validator makes the same
  // constraint authoritative in the reactive form so the request is never sent.
  private readonly logRangeValidator = (control: AbstractControl): ValidationErrors | null => {
    const value = control.value as string;
    if (!value) {
      return null;
    }
    const date = dayjs(value);
    if (!date.isValid()) {
      return null;
    }
    if (date.isAfter(dayjs())) {
      return {afterNow: true};
    }
    const windowSeconds = this.organization()?.subscriptionLimits.logQueryWindowSeconds;
    if (windowSeconds && date.isBefore(dayjs().subtract(windowSeconds, 'second'))) {
      return {beforeWindow: true};
    }
    return null;
  };

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

    // The window validator depends on the organization, which loads asynchronously and
    // is not reactive inside Angular validators, so re-validate once it is available.
    effect(() => {
      this.organization();
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
      if (this.form.controls.from.invalid || this.form.controls.to.invalid) {
        return;
      }
      this.router.navigate([], {
        relativeTo: this.route,
        queryParams: {
          from: dateTimeLocalToISO(values.from),
          to: dateTimeLocalToISO(values.to),
          filter: values.filter || null,
        },
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
