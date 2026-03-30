import {OverlayModule} from '@angular/cdk/overlay';
import {Component, computed, ElementRef, inject, signal, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {DeploymentWithLatestRevision} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faServer} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, map, of, switchMap} from 'rxjs';
import {DeploymentLogsService} from '../../services/deployment-logs.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {DeploymentLogsTableComponent} from '../deployment-status-modal/deployment-logs-table.component';
import {DeploymentStatusTableComponent} from '../deployment-status-modal/deployment-status-table.component';
import {DeploymentAppNameComponent} from '../deployment-target-card/deployment-app-name.component';
import {DeploymentTargetLogsTableComponent} from '../deployment-target-status-modal/deployment-target-logs-table.component';

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
    RouterLink,
  ],
})
export class DeploymentTargetDetailComponent {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly deploymentLogsService = inject(DeploymentLogsService);

  protected readonly faServer = faServer;
  protected readonly faChevronDown = faChevronDown;

  protected readonly targetDropdown = signal(false);
  protected targetDropdownWidth = 0;
  protected readonly targetDropdownTrigger = viewChild.required<ElementRef<HTMLElement>>('targetDropdownTrigger');

  protected readonly deploymentDropdown = signal(false);
  protected deploymentDropdownWidth = 0;
  protected readonly deploymentDropdownTrigger =
    viewChild.required<ElementRef<HTMLElement>>('deploymentDropdownTrigger');

  protected readonly resourceDropdown = signal(false);
  protected resourceDropdownWidth = 0;
  protected readonly resourceDropdownTrigger = viewChild<ElementRef<HTMLElement>>('resourceDropdownTrigger');

  protected readonly deploymentTargetId = toSignal(this.route.paramMap.pipe(map((p) => p.get('deploymentTargetId')!)));
  protected readonly deploymentId = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('deploymentId'))));
  protected readonly resource = toSignal(this.route.queryParamMap.pipe(map((p) => p.get('resource'))));

  protected readonly deploymentTargets = toSignal(this.deploymentTargetsService.list(), {initialValue: []});
  protected readonly deploymentTarget = toSignal(
    combineLatest([
      this.deploymentTargetsService.list(),
      this.route.paramMap.pipe(map((p) => p.get('deploymentTargetId')!)),
    ]).pipe(map(([targets, id]) => targets.find((t) => t.id === id)))
  );

  protected readonly selectedDeployment = computed(() => {
    const id = this.deploymentId();
    return id ? this.deploymentTarget()?.deployments?.find((d) => d.id === id) : undefined;
  });

  protected readonly resources = toSignal(
    this.route.queryParamMap.pipe(
      map((p) => p.get('deploymentId')),
      switchMap((id) => (id ? this.deploymentLogsService.getResources(id) : of(null)))
    )
  );

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
    this.deploymentDropdown.set(false);
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {deploymentId: deployment?.id ?? null, resource: null},
      queryParamsHandling: 'merge',
    });
  }

  protected selectResource(resource: string | null) {
    this.resourceDropdown.set(false);
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: {resource},
      queryParamsHandling: 'merge',
    });
  }
}
