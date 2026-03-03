import {AsyncPipe} from '@angular/common';
import {Component, inject, OnDestroy, OnInit} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';
import {catchError, combineLatestWith, first, map, of, shareReplay, Subject, switchMap, takeUntil} from 'rxjs';
import {ArtifactsByCustomerCardComponent} from '../../artifacts/artifacts-by-customer-card/artifacts-by-customer-card.component';
import {DeploymentTargetDashboardCardComponent} from '../../deployments/deployment-target-card/deployment-target-dashboard-card.component';
import {DeploymentTargetViewData} from '../../deployments/deployment-targets.component';
import {DashboardService} from '../../services/dashboard.service';
import {DeploymentTargetsMetricsService} from '../../services/deployment-target-metrics.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleDashboardCardComponent} from '../../support-bundles/dashboard-card/support-bundle-dashboard-card.component';
import {SupportBundle} from '../../types/support-bundle';

@Component({
  selector: 'app-dashboard',
  imports: [
    AsyncPipe,
    ArtifactsByCustomerCardComponent,
    DeploymentTargetDashboardCardComponent,
    SupportBundleDashboardCardComponent,
  ],
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit, OnDestroy {
  private readonly destroyed$ = new Subject<void>();
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);
  private readonly dashboardService = inject(DashboardService);
  private readonly supportBundlesService = inject(SupportBundlesService);
  protected readonly artifactsByCustomer$ = this.dashboardService.getArtifactsByCustomer().pipe(shareReplay(1));
  protected readonly supportBundlesByCustomer$ = this.supportBundlesService.list().pipe(
    map((bundles) => {
      const grouped = new Map<string, {customerName: string; bundles: SupportBundle[]}>();
      for (const bundle of bundles) {
        const existing = grouped.get(bundle.customerOrganizationId);
        if (existing) {
          existing.bundles.push(bundle);
        } else {
          grouped.set(bundle.customerOrganizationId, {
            customerName: bundle.customerOrganizationName,
            bundles: [bundle],
          });
        }
      }
      return Array.from(grouped.values()).sort((a, b) => a.customerName.localeCompare(b.customerName));
    }),
    catchError(() => of([])),
    shareReplay(1)
  );
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly deploymentTargetMetricsService = inject(DeploymentTargetsMetricsService);
  protected readonly deploymentTargets$ = this.deploymentTargetsService
    .poll()
    .pipe(takeUntil(this.destroyed$), shareReplay(1));
  protected readonly deploymentTargetMetrics$ = this.deploymentTargetMetricsService.poll().pipe(
    takeUntil(this.destroyed$),
    catchError(() => of([]))
  );
  protected readonly deploymentTargetWithMetrics$ = this.deploymentTargets$.pipe(
    combineLatestWith(this.deploymentTargetMetrics$),
    map(([deploymentTargets, deploymentTargetMetrics]) => {
      return deploymentTargets.map((dt) => {
        return {
          ...dt,
          metrics: deploymentTargetMetrics.find((x) => x.id === dt.id),
          // TODO deduplicate
        } as DeploymentTargetViewData;
      });
    })
  );

  ngOnInit() {
    if (this.route.snapshot.queryParams?.['from'] === 'login') {
      this.artifactsByCustomer$
        .pipe(
          takeUntil(this.destroyed$),
          combineLatestWith(this.deploymentTargetsService.list()),
          first(),
          switchMap(([artifacts, dts]) => {
            if (artifacts.length === 0 && dts.length === 0) {
              return this.router.navigate(['tutorials']);
            } else {
              return this.router.navigate([this.router.url]); // remove query param
            }
          })
        )
        .subscribe();
    } else if (this.route.snapshot.queryParams?.['from'] === 'new-org') {
      this.toast.success('New organization created successfully');
      this.router.navigate([this.router.url]); // remove query param
    }
  }

  ngOnDestroy() {
    this.destroyed$.next();
    this.destroyed$.complete();
  }
}
