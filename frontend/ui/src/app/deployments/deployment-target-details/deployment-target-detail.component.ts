import { OverlayModule } from '@angular/cdk/overlay';
import { Component, computed, ElementRef, inject, signal, viewChild } from '@angular/core';
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop';
import { FormBuilder, ReactiveFormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { DeploymentWithLatestRevision } from '@distr-sh/distr-sdk';
import { FaIconComponent } from '@fortawesome/angular-fontawesome';
import { faChevronDown, faDownload, faFilterCircleXmark, faPlay, faServer } from '@fortawesome/free-solid-svg-icons';
import { combineLatest, debounceTime, map, of, switchMap } from 'rxjs';
import { DeploymentLogsService } from '../../services/deployment-logs.service';
import { DeploymentTargetsService } from '../../services/deployment-targets.service';
import { DeploymentLogsTableComponent } from '../deployment-status-modal/deployment-logs-table.component';
import { DeploymentStatusTableComponent } from '../deployment-status-modal/deployment-status-table.component';
import { DeploymentAppNameComponent } from '../deployment-target-card/deployment-app-name.component';
import { DeploymentTargetLogsTableComponent } from '../deployment-target-status-modal/deployment-target-logs-table.component';
import dayjs from 'dayjs';

@Component({
  selector: 'app-deployment-target-detail',
  templateUrl: './deployment-target-detail.component.html',
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
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faServer = faServer;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faDownload = faDownload;
  protected readonly faFilterCircleXmark = faFilterCircleXmark;
  protected readonly faPlay = faPlay;

  protected readonly targetDropdown = signal(false);
  protected targetDropdownWidth = 0;
  private readonly targetDropdownTrigger = viewChild.required<ElementRef<HTMLElement>>('targetDropdownTrigger');

  protected readonly deploymentDropdown = signal(false);
  protected deploymentDropdownWidth = 0;
  private readonly deploymentDropdownTrigger = viewChild.required<ElementRef<HTMLElement>>('deploymentDropdownTrigger');

  protected readonly resourceDropdown = signal(false);
  protected resourceDropdownWidth = 0;
  private readonly resourceDropdownTrigger = viewChild<ElementRef<HTMLElement>>('resourceDropdownTrigger');

  private readonly deploymentTargetId$ = this.route.paramMap.pipe(map((p) => p.get('deploymentTargetId')!));
  protected readonly deploymentTargetId = toSignal(this.deploymentTargetId$);
  private readonly deploymentId$ = this.route.queryParamMap.pipe(map((p) => p.get('deploymentId')));
  protected readonly deploymentId = toSignal(this.deploymentId$);
  private readonly resource$ = this.route.queryParamMap.pipe(map((p) => p.get('resource')));
  protected readonly resource = toSignal(this.resource$);
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

  protected readonly live = computed(() => !this.after() && !this.before())

  private readonly deploymentTargets$ = this.deploymentTargetsService.list();
  protected readonly deploymentTargets = toSignal(this.deploymentTargets$, { initialValue: [] });
  protected readonly selectedDeploymentTarget = toSignal(
    combineLatest([this.deploymentTargets$, this.deploymentTargetId$]).pipe(
      map(([targets, id]) => targets.find((t) => t.id === id))
    )
  );

  protected readonly selectedDeployment = computed(() => {
    const id = this.deploymentId();
    return id ? this.selectedDeploymentTarget()?.deployments?.find((d) => d.id === id) : undefined;
  });

  protected readonly resources = toSignal(
    this.route.queryParamMap.pipe(
      map((p) => p.get('deploymentId')),
      switchMap((id) => (id ? this.deploymentLogsService.getResources(id) : of(null)))
    )
  );

  protected readonly form = this.fb.group({ from: '', to: '', filter: '' });

  private readonly deploymentTargetLogsTable = viewChild(DeploymentTargetLogsTableComponent);
  private readonly deploymentStatusTable = viewChild(DeploymentStatusTableComponent);
  private readonly deploymentLogsTable = viewChild(DeploymentLogsTableComponent);

  constructor() {
    this.route.queryParamMap.pipe(takeUntilDestroyed()).subscribe((params) => {
      this.form.patchValue(
        {
          from: isoToDateTimeLocal(params.get('from')),
          to: isoToDateTimeLocal(params.get('to')),
          filter: params.get('filter') ?? '',
        },
        { emitEvent: false }
      );
    });

    this.form.valueChanges.pipe(takeUntilDestroyed(), debounceTime(300)).subscribe((values) => {
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
    this.form.patchValue({ filter: '' });
    this.deploymentDropdown.set(false);
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { deploymentId: deployment?.id ?? null, resource: null },
      queryParamsHandling: 'merge',
    });
  }

  protected selectResource(resource: string | undefined) {
    this.form.patchValue({ filter: '' });
    this.resourceDropdown.set(false);
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { resource: resource ?? null },
      queryParamsHandling: 'merge',
    });
  }

  protected resetAllFilters() {
    this.form.reset();
  }

  protected resetDateFilters() {
    this.form.patchValue({ from: '', to: '' });
  }

  protected export() {
    // Only one of the tables is shown at any given time, so it's fine to call export on all of them
    this.deploymentTargetLogsTable()?.export();
    this.deploymentStatusTable()?.export();
    this.deploymentLogsTable()?.export();
  }
}

function dateTimeLocalToISO(dateTimeLocal: string | null | undefined): string | null {
  return dateTimeLocal ? dayjs(dateTimeLocal).toISOString() : null;
}

function isoToDateTimeLocal(iso: string | null | undefined): string {
  return iso ? dayjs(iso).local().format('YYYY-MM-DDTHH:mm') : '';
}
