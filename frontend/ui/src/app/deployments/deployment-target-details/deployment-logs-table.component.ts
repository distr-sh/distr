import {Component, computed, inject, input, viewChild} from '@angular/core';
import {map, Observable} from 'rxjs';
import {
  TimeseriesEntry,
  TimeseriesExporter,
  TimeseriesSource,
  TimeseriesTableComponent,
} from '../../components/timeseries-table/timeseries-table.component';
import {DeploymentLogsService} from '../../services/deployment-logs.service';
import {DeploymentLogRecord} from '../../types/deployment-log-record';
import {OrderDirection} from '../../types/timeseries-options';

const ansiEscapePattern = /\u001b[^m]*m/g;

const RESOURCE_COLORS = [
  '#3b82f6', // blue
  '#10b981', // emerald
  '#f59e0b', // amber
  '#8b5cf6', // violet
  '#06b6d4', // cyan
  '#f43f5e', // rose
  '#84cc16', // lime
  '#ec4899', // pink
  '#14b8a6', // teal
  '#f97316', // orange
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

class LogsTimeseriesSource implements TimeseriesSource {
  public readonly batchSize = 50;

  constructor(
    private readonly svc: DeploymentLogsService,
    private readonly deploymentId: string,
    private readonly resources: string[],
    private readonly order: OrderDirection,
    private readonly after?: Date,
    private readonly before?: Date,
    private readonly filter?: string
  ) {}

  load(): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentId, this.resources, {
        limit: this.batchSize,
        after: this.after,
        before: this.before,
        filter: this.filter,
        order: this.order,
      })
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }

  loadAfter(after: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentId, this.resources, {limit: this.batchSize, after, filter: this.filter, order: this.order})
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }

  loadBefore(before: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentId, this.resources, {limit: this.batchSize, before, filter: this.filter, order: this.order})
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }
}

@Component({
  selector: 'app-deployment-logs-table',
  template: `<app-timeseries-table
    [source]="source()"
    [exporter]="exporter"
    [live]="live()"
    [orderDirection]="orderDirection()"
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
  public readonly orderDirection = input(OrderDirection.DESC);

  protected readonly live = computed(() => !this.after() && !this.before());

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
      new LogsTimeseriesSource(
        this.svc,
        this.deploymentId(),
        this.resources(),
        this.orderDirection(),
        this.after(),
        this.before(),
        this.filter()
      )
  );

  protected readonly exporter: TimeseriesExporter = {
    export: () => this.svc.export(this.deploymentId(), this.resources()),
    getFileName: () => `${this.resources().join('_')}.log`,
  };

  private readonly table = viewChild.required(TimeseriesTableComponent);

  public export() {
    this.table().exportData();
  }
}
