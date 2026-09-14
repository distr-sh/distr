import {OverlayModule} from '@angular/cdk/overlay';
import {PercentPipe} from '@angular/common';
import {Component, computed, inject, input, signal} from '@angular/core';
import {rxResource} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircleQuestion, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {switchMap, timer} from 'rxjs';
import {RelativeDatePipe} from '../../../util/dates';
import {isStale} from '../../../util/model';
import {BytesPipe} from '../../../util/units';
import {SpinnerComponent} from '../../components/spinner/spinner.component';
import {DeploymentTargetsMetricsService} from '../../services/deployment-target-metrics.service';
import {DeploymentResourceMetric} from '../../types/deployment-target-metrics';

interface ResourceRow {
  metric: DeploymentResourceMetric;
  cpuUsageRatio?: number;
  memoryUsageRatio?: number;
}

interface ResourceGroup {
  resource: string;
  rows: ResourceRow[];
}

const staleThreshold = dayjs.duration(2, 'minutes');

@Component({
  selector: 'app-deployment-resource-metrics',
  templateUrl: './deployment-resource-metrics.component.html',
  imports: [OverlayModule, PercentPipe, BytesPipe, SpinnerComponent, FaIconComponent, RelativeDatePipe],
})
export class DeploymentResourceMetricsComponent {
  public readonly deploymentId = input.required<string>();

  protected readonly faCircleQuestion = faCircleQuestion;
  protected readonly faTriangleExclamation = faTriangleExclamation;
  protected readonly cpuHelpHovered = signal(false);

  private readonly metricsService = inject(DeploymentTargetsMetricsService);

  protected readonly metrics = rxResource({
    params: () => ({deploymentId: this.deploymentId()}),
    stream: ({params}) =>
      timer(0, 30_000).pipe(switchMap(() => this.metricsService.getDeploymentMetrics(params.deploymentId))),
  });

  protected readonly createdAt = computed(() => this.metrics.value()?.createdAt);
  protected readonly stale = computed(() => {
    const metrics = this.metrics.value();
    return metrics !== undefined && isStale(metrics, staleThreshold);
  });

  protected readonly groups = computed<ResourceGroup[]>(() => {
    const groups = new Map<string, ResourceGroup>();
    for (const metric of this.metrics.value()?.resources ?? []) {
      let group = groups.get(metric.resource);
      if (!group) {
        group = {resource: metric.resource, rows: []};
        groups.set(metric.resource, group);
      }
      group.rows.push({
        metric,
        cpuUsageRatio: metric.cpuLimitMillis ? metric.cpuUsageMillis / metric.cpuLimitMillis : undefined,
        memoryUsageRatio: metric.memoryLimitBytes ? metric.memoryBytes / metric.memoryLimitBytes : undefined,
      });
    }
    return [...groups.values()];
  });
}
