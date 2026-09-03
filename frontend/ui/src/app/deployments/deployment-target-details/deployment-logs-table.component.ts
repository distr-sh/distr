import {Component, computed, inject, input, viewChild} from '@angular/core';
import {
  RangeTimeseriesSource,
  TimeseriesEntry,
  TimeseriesExporter,
} from '../../components/timeseries-table/timeseries-source';
import {TimeseriesTableComponent} from '../../components/timeseries-table/timeseries-table.component';
import {DeploymentLogsService} from '../../services/deployment-logs.service';
import {DeploymentLogRecord} from '../../types/deployment-log-record';
import {OrderDirection} from '../../types/timeseries-options';

const ansiEscapePattern = /\u001b[^m]*m/g;

const RESOURCE_COLORS = [
  'text-blue-600 dark:text-blue-400',
  'text-emerald-600 dark:text-emerald-400',
  'text-amber-600 dark:text-amber-400',
  'text-violet-600 dark:text-violet-400',
  'text-cyan-600 dark:text-cyan-400',
  'text-rose-600 dark:text-rose-400',
  'text-lime-700 dark:text-lime-400',
  'text-pink-600 dark:text-pink-400',
  'text-teal-600 dark:text-teal-400',
  'text-orange-600 dark:text-orange-400',
];

function logRecordToTimeseriesEntry(record: DeploymentLogRecord): TimeseriesEntry {
  return {
    id: record.id,
    date: record.timestamp,
    status: record.severity,
    detail: record.body.trim().replace(ansiEscapePattern, ''),
    resource: record.resource,
  };
}

@Component({
  selector: 'app-deployment-logs-table',
  template: `<app-timeseries-table
    [source]="source()"
    [exporter]="exporter"
    [resourceColorMap]="resourceColorMap()" />`,
  imports: [TimeseriesTableComponent],
})
export class DeploymentLogsTableComponent {
  private readonly svc = inject(DeploymentLogsService);

  public readonly deploymentId = input.required<string>();
  public readonly resources = input.required<string[]>();
  public readonly after = input<Date>();
  public readonly before = input<Date>();
  public readonly filter = input<string>();
  public readonly orderDirection = input<OrderDirection>('DESC');

  protected readonly resourceColorMap = computed(() => {
    const resources = this.resources();
    const map: Record<string, string> = {};
    for (let i = 0; i < resources.length; i++) {
      map[resources[i]] = RESOURCE_COLORS[i % RESOURCE_COLORS.length];
    }
    return map;
  });

  protected readonly source = computed(
    () =>
      new RangeTimeseriesSource(
        (options) => this.svc.get(this.deploymentId(), this.resources(), options),
        logRecordToTimeseriesEntry,
        {order: this.orderDirection(), after: this.after(), before: this.before(), filter: this.filter()}
      )
  );

  protected readonly exporter: TimeseriesExporter = {
    export: () =>
      this.svc.export(this.deploymentId(), this.resources(), {
        after: this.after(),
        before: this.before(),
        filter: this.filter(),
      }),
    getFileName: () => `${this.resources().join('_')}.log`,
  };

  private readonly table = viewChild.required(TimeseriesTableComponent);

  public export() {
    this.table().exportData();
  }
}
