import {Component, computed, inject, input, viewChild} from '@angular/core';
import {DeploymentRevisionStatus} from '@distr-sh/distr-sdk';
import {
  RangeTimeseriesSource,
  TimeseriesEntry,
  TimeseriesExporter,
} from '../../components/timeseries-table/timeseries-source';
import {TimeseriesTableComponent} from '../../components/timeseries-table/timeseries-table.component';
import {DeploymentStatusService} from '../../services/deployment-status.service';
import {OrderDirection} from '../../types/timeseries-options';

function statusToTimeseriesEntry(record: DeploymentRevisionStatus): TimeseriesEntry {
  return {id: record.id, date: record.createdAt!, status: record.type, detail: record.message};
}

@Component({
  selector: 'app-deployment-status-table',
  template: `<app-timeseries-table [source]="source()" [exporter]="exporter" />`,
  imports: [TimeseriesTableComponent],
})
export class DeploymentStatusTableComponent {
  private readonly svc = inject(DeploymentStatusService);

  public readonly deploymentId = input.required<string>();
  public readonly after = input<Date>();
  public readonly before = input<Date>();
  public readonly filter = input<string>();
  public readonly orderDirection = input<OrderDirection>('DESC');

  protected readonly source = computed(
    () =>
      new RangeTimeseriesSource(
        (options) => this.svc.getStatuses(this.deploymentId(), options),
        statusToTimeseriesEntry,
        {order: this.orderDirection(), after: this.after(), before: this.before(), filter: this.filter()}
      )
  );

  protected readonly exporter: TimeseriesExporter = {
    export: () =>
      this.svc.export(this.deploymentId(), {
        after: this.after(),
        before: this.before(),
        filter: this.filter(),
      }),
    getFileName: () => `deployment_status.log`,
  };

  private readonly table = viewChild.required(TimeseriesTableComponent);

  public export() {
    this.table().exportData();
  }
}
