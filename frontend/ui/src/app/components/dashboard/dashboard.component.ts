import {Component, inject, OnInit} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faSpinner} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatestWith, first, map, of, shareReplay, switchMap} from 'rxjs';
import {ArtifactsByCustomerCardComponent} from '../../artifacts/artifacts-by-customer-card/artifacts-by-customer-card.component';
import {DeploymentTargetDashboardCardComponent} from '../../deployments/deployment-target-card/deployment-target-dashboard-card.component';
import {DeploymentTargetViewData} from '../../deployments/deployment-targets.component';
import {DashboardService} from '../../services/dashboard.service';
import {DeploymentTargetsMetricsService} from '../../services/deployment-target-metrics.service';
import {DeploymentTargetsService} from '../../services/deployment-targets.service';
import {FeatureFlagService} from '../../services/feature-flag.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleDashboardCardComponent} from '../../support-bundles/dashboard-card/support-bundle-dashboard-card.component';
import {SupportBundle} from '../../types/support-bundle';

@Component({
  selector: 'app-dashboard',
  imports: [
    ArtifactsByCustomerCardComponent,
    DeploymentTargetDashboardCardComponent,
    FaIconComponent,
    SupportBundleDashboardCardComponent,
  ],
  templateUrl: './dashboard.component.html',
})
export class DashboardComponent implements OnInit {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);
  private readonly dashboardService = inject(DashboardService);
  private readonly featureFlags = inject(FeatureFlagService);
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly deploymentTargetsService = inject(DeploymentTargetsService);
  private readonly deploymentTargetMetricsService = inject(DeploymentTargetsMetricsService);

  private readonly artifactsByCustomer$ = this.dashboardService.getArtifactsByCustomer().pipe(shareReplay(1));
  protected readonly artifactsByCustomer = toSignal(this.artifactsByCustomer$);

  protected readonly supportBundlesByCustomer = toSignal(
    this.featureFlags.isSupportBundlesEnabled$.pipe(
      switchMap((enabled) => (enabled ? this.supportBundlesService.list() : of([]))),
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
      catchError(() => of([]))
    ),
    {initialValue: []}
  );

  private readonly deploymentTargets$ = this.deploymentTargetsService.poll().pipe(takeUntilDestroyed(), shareReplay(1));
  private readonly deploymentTargetMetrics$ = this.deploymentTargetMetricsService.poll().pipe(
    takeUntilDestroyed(),
    catchError(() => of([]))
  );

  protected readonly deploymentTargetWithMetrics = toSignal(
    this.deploymentTargets$.pipe(
      combineLatestWith(this.deploymentTargetMetrics$),
      map(([deploymentTargets, deploymentTargetMetrics]) =>
        deploymentTargets.map(
          (dt) =>
            ({
              ...dt,
              metrics: deploymentTargetMetrics.find((x) => x.deploymentTargetId === dt.id),
            }) as DeploymentTargetViewData
        )
      )
    )
  );

  protected readonly faSpinner = faSpinner;

  ngOnInit() {
    if (this.route.snapshot.queryParams?.['from'] === 'login') {
      this.artifactsByCustomer$
        .pipe(
          combineLatestWith(this.deploymentTargetsService.list()),
          first(),
          switchMap(([artifacts, dts]) => {
            if (artifacts.length === 0 && dts.length === 0) {
              return this.router.navigate(['tutorials']);
            } else {
              return this.router.navigate([this.router.url]);
            }
          })
        )
        .subscribe();
    } else if (this.route.snapshot.queryParams?.['from'] === 'new-org') {
      this.toast.success('New organization created successfully');
      this.router.navigate([this.router.url]);
    }
  }
}
