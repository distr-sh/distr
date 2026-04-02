import {Component, computed, inject, input, viewChild} from '@angular/core';
import {DeploymentRevisionStatus} from '@distr-sh/distr-sdk';
import {map, Observable} from 'rxjs';
import {
  TimeseriesEntry,
  TimeseriesExporter,
  TimeseriesSource,
  TimeseriesTableComponent,
} from '../../components/timeseries-table/timeseries-table.component';
import {DeploymentStatusService} from '../../services/deployment-status.service';

function statusToTimeseriesEntry(record: DeploymentRevisionStatus): TimeseriesEntry {
  return {id: record.id, date: record.createdAt!, status: record.type, detail: record.message};
}

class LogsTimeseriesSource implements TimeseriesSource {
  public readonly batchSize = 25;

  constructor(
    private readonly svc: DeploymentStatusService,
    private readonly deploymentId: string,
    private readonly after?: Date,
    private readonly before?: Date,
    private readonly filter?: string
  ) {}

  load(): Observable<TimeseriesEntry[]> {
    return this.svc
      .getStatuses(this.deploymentId, {
        limit: this.batchSize,
        after: this.after,
        before: this.before,
        filter: this.filter,
      })
      .pipe(map((logs) => logs.map(statusToTimeseriesEntry)));
  }

  loadAfter(after: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .getStatuses(this.deploymentId, {limit: this.batchSize, after, filter: this.filter})
      .pipe(map((logs) => logs.map(statusToTimeseriesEntry)));
  }

  loadBefore(before: Date): Observable<TimeseriesEntry[]> {
    return this.svc
      .getStatuses(this.deploymentId, {limit: this.batchSize, before, filter: this.filter})
      .pipe(map((logs) => logs.map(statusToTimeseriesEntry)));
  }
}

@Component({
  selector: 'app-deployment-status-table',
  template: `<app-timeseries-table
    [source]="source()"
    [exporter]="exporter"
    [live]="live()"
    [newestFirst]="newestFirst()" />`,
  imports: [TimeseriesTableComponent],
})
export class DeploymentStatusTableComponent {
  private readonly svc = inject(DeploymentStatusService);

  public readonly deploymentId = input.required<string>();
  public readonly after = input<Date>();
  public readonly before = input<Date>();
  public readonly filter = input<string>();
  public readonly newestFirst = input(true);

  protected readonly live = computed(() => !this.after() && !this.before());

  protected readonly source = computed(
    () => new LogsTimeseriesSource(this.svc, this.deploymentId(), this.after(), this.before(), this.filter())
  );

  protected readonly exporter: TimeseriesExporter = {
    export: () => this.svc.export(this.deploymentId()),
    getFileName: () => `deployment_status.log`,
  };

  private readonly table = viewChild.required(TimeseriesTableComponent);

  public export() {
    this.table().exportData();
  }
}
