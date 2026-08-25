import {OverlayModule} from '@angular/cdk/overlay';
import {PercentPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, input, signal} from '@angular/core';
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

interface ResourceGroup {
  resource: string;
  containers: DeploymentResourceMetric[];
}

const staleThreshold = dayjs.duration(2, 'minutes');

@Component({
  selector: 'app-deployment-resource-metrics',
  templateUrl: './deployment-resource-metrics.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
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
    for (const container of this.metrics.value()?.resources ?? []) {
      let group = groups.get(container.resource);
      if (!group) {
        group = {resource: container.resource, containers: []};
        groups.set(container.resource, group);
      }
      group.containers.push(container);
    }
    return [...groups.values()];
  });

  protected cpuUsageRatio(usageMillis: number, limitMillis: number | undefined): number | undefined {
    return limitMillis ? usageMillis / limitMillis : undefined;
  }

  protected memoryUsageRatio(usageBytes: number, limitBytes: number | undefined): number | undefined {
    return limitBytes ? usageBytes / limitBytes : undefined;
  }
}
