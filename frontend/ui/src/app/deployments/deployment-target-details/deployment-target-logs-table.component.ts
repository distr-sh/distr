import {Component, computed, inject, input, viewChild} from '@angular/core';
import {
  RangeTimeseriesSource,
  TimeseriesEntry,
  TimeseriesExporter,
} from '../../components/timeseries-table/timeseries-source';
import {TimeseriesTableComponent} from '../../components/timeseries-table/timeseries-table.component';
import {DeploymentTargetLogsService} from '../../services/deployment-target-logs.service';
import {DeploymentTargetLogRecord} from '../../types/deployment-target-log-record';
import {OrderDirection} from '../../types/timeseries-options';

function logRecordToTimeseriesEntry(record: DeploymentTargetLogRecord): TimeseriesEntry {
  return {id: record.id, date: record.timestamp, status: record.severity, detail: record.body.trim()};
}

@Component({
  selector: 'app-deployment-target-logs-table',
  template: `<app-timeseries-table [source]="source()" [exporter]="exporter" />`,
  imports: [TimeseriesTableComponent],
})
export class DeploymentTargetLogsTableComponent {
  private readonly svc = inject(DeploymentTargetLogsService);

  public readonly deploymentTargetId = input.required<string>();
  public readonly after = input<Date>();
  public readonly before = input<Date>();
  public readonly filter = input<string>();
  public readonly orderDirection = input<OrderDirection>('DESC');

  protected readonly source = computed(
    () =>
      new RangeTimeseriesSource(
        (options) => this.svc.get(this.deploymentTargetId(), options),
        logRecordToTimeseriesEntry,
        {order: this.orderDirection(), after: this.after(), before: this.before(), filter: this.filter()}
      )
  );

  protected readonly exporter: TimeseriesExporter = {
    export: () =>
      this.svc.export(this.deploymentTargetId(), {
        after: this.after(),
        before: this.before(),
        filter: this.filter(),
      }),
    getFileName: () => 'agent.log',
  };

  private readonly table = viewChild.required(TimeseriesTableComponent);

  public export() {
    this.table().exportData();
  }
}
