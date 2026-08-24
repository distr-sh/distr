import {OverlayModule} from '@angular/cdk/overlay';
import {PercentPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, input, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faExclamation, faHardDrive, faTriangleExclamation} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {RelativeDatePipe} from '../../../util/dates';
import {isStale} from '../../../util/model';
import {BytesPipe} from '../../../util/units';
import {StatusDotDirective} from '../../components/status-dot';
import {DeploymentTargetLatestMetrics} from '../../types/deployment-target-metrics';

// Agents report metrics every 30 seconds, so one missed report must not mark them as outdated yet.
const metricsStaleThreshold = dayjs.duration({minutes: 2});

@Component({
  selector: 'app-deployment-target-metrics',
  templateUrl: './deployment-target-metrics.component.html',
  imports: [OverlayModule, BytesPipe, PercentPipe, FaIconComponent, StatusDotDirective, RelativeDatePipe],
  changeDetection: ChangeDetectionStrategy.Eager,
  styleUrls: ['./deployment-target-metrics.component.scss'],
})
export class DeploymentTargetMetricsComponent {
  public readonly metrics = input.required<DeploymentTargetLatestMetrics>();
  protected readonly hovered = signal(false);
  protected readonly anyDiskWarning = computed(() =>
    this.metrics().diskMetrics?.some((disk) => disk.bytesUsed / disk.bytesTotal > 0.75)
  );
  protected readonly outdated = computed(() => isStale(this.metrics(), metricsStaleThreshold));

  protected readonly faHardDrive = faHardDrive;
  protected readonly faExclamation = faExclamation;
  protected readonly faTriangleExclamation = faTriangleExclamation;

  protected getUsageDegrees(value: number | undefined): string {
    return (360 * (value ?? 0)).toFixed() + 'deg';
  }
}
