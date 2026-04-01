import {Component, computed, inject, input, viewChild} from '@angular/core';
import {map, Observable} from 'rxjs';
import {
  TimeseriesEntry,
  TimeseriesExporter,
  TimeseriesSource,
  TimeseriesTableComponent,
} from '../../components/timeseries-table.component';
import {DeploymentLogsService} from '../../services/deployment-logs.service';
import {DeploymentLogRecord} from '../../types/deployment-log-record';

const ansiEscapePattern = /\u001b[^m]*m/g;

function logRecordToTimeseriesEntry(record: DeploymentLogRecord): TimeseriesEntry {
  return {date: record.timestamp, status: record.severity, detail: record.body.trim().replace(ansiEscapePattern, '')};
}

class LogsTimeseriesSource implements TimeseriesSource {
  public readonly batchSize = 25;

  constructor(
    private readonly svc: DeploymentLogsService,
    private readonly deploymentId: string,
    private readonly resource: string,
    private readonly after?: Date,
    private readonly before?: Date,
    private readonly filter?: string
  ) {}

  load(): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentId, this.resource, {
        limit: this.batchSize,
        after: this.after,
        before: this.before,
        filter: this.filter,
      })
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }

  loadAfter(after: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentId, this.resource, {limit: this.batchSize, after, filter: this.filter})
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }

  loadBefore(before: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentId, this.resource, {limit: this.batchSize, before, filter: this.filter})
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }
}

@Component({
  selector: 'app-deployment-logs-table',
  template: `<app-timeseries-table
    [source]="source()"
    [exporter]="exporter"
    [live]="live()"
    [newestFirst]="newestFirst()" />`,
  imports: [TimeseriesTableComponent],
})
export class DeploymentLogsTableComponent {
  private readonly svc = inject(DeploymentLogsService);

  public readonly deploymentId = input.required<string>();
  public readonly resource = input.required<string>();
  public readonly after = input<Date>();
  public readonly before = input<Date>();
  public readonly filter = input<string>();
  public readonly newestFirst = input(true);

  protected readonly live = computed(() => !this.after() && !this.before());

  protected readonly source = computed(
    () =>
      new LogsTimeseriesSource(
        this.svc,
        this.deploymentId(),
        this.resource(),
        this.after(),
        this.before(),
        this.filter()
      )
  );

  protected readonly exporter: TimeseriesExporter = {
    export: () => this.svc.export(this.deploymentId(), this.resource()),
    getFileName: () => `${this.resource()}.log`,
  };

  private readonly table = viewChild.required(TimeseriesTableComponent);

  public export() {
    this.table().exportData();
  }
}
