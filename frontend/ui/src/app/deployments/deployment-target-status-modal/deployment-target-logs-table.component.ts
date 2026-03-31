import {Component, computed, inject, input, viewChild} from '@angular/core';
import {map, Observable} from 'rxjs';
import {
  TimeseriesEntry,
  TimeseriesExporter,
  TimeseriesSource,
  TimeseriesTableComponent,
} from '../../components/timeseries-table.component';
import {DeploymentTargetLogsService} from '../../services/deployment-target-logs.service';
import {DeploymentTargetLogRecord} from '../../types/deployment-target-log-record';

function logRecordToTimeseriesEntry(record: DeploymentTargetLogRecord): TimeseriesEntry {
  return {id: record.id, date: record.timestamp, status: record.severity, detail: record.body.trim()};
}

class LogsTimeseriesSource implements TimeseriesSource {
  public readonly batchSize = 25;

  constructor(
    private readonly svc: DeploymentTargetLogsService,
    private readonly deploymentTargetId: string,
    private readonly after?: Date,
    private readonly before?: Date,
    private readonly filter?: string
  ) {}

  load(): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentTargetId, {
        limit: this.batchSize,
        after: this.after,
        before: this.before,
        filter: this.filter,
      })
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }

  loadAfter(after: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentTargetId, {limit: this.batchSize, after, filter: this.filter})
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }

  loadBefore(before: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .get(this.deploymentTargetId, {limit: this.batchSize, before, filter: this.filter})
      .pipe(map((logs) => logs.map(logRecordToTimeseriesEntry)));
  }
}

@Component({
  selector: 'app-deployment-target-logs-table',
  template: `<app-timeseries-table
    [source]="source()"
    [exporter]="exporter"
    [live]="live()"
    [newestFirst]="newestFirst()" />`,
  imports: [TimeseriesTableComponent],
})
export class DeploymentTargetLogsTableComponent {
  private readonly svc = inject(DeploymentTargetLogsService);

  public readonly deploymentTargetId = input.required<string>();
  public readonly after = input<Date>();
  public readonly before = input<Date>();
  public readonly filter = input<string>();
  public readonly newestFirst = input(true);

  protected readonly live = computed(() => !this.after() && !this.before());

  protected readonly source = computed(
    () => new LogsTimeseriesSource(this.svc, this.deploymentTargetId(), this.after(), this.before(), this.filter())
  );

  protected readonly exporter: TimeseriesExporter = {
    export: () => this.svc.export(this.deploymentTargetId()),
    getFileName: () => 'agent.log',
  };

  private readonly table = viewChild.required(TimeseriesTableComponent);

  public export() {
    this.table().exportData();
  }
}
