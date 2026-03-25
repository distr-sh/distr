import {OverlayModule} from '@angular/cdk/overlay';
import {NgStyle, PercentPipe} from '@angular/common';
import {Component, computed, input, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faExclamation, faHardDrive} from '@fortawesome/free-solid-svg-icons';
import {BytesPipe} from '../../../util/units';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {StatusDotDirective} from '../../components/status-dot';
import {DeploymentTargetLatestMetrics} from '../../types/deployment-target-metrics';

@Component({
  selector: 'app-deployment-target-metrics',
  templateUrl: './deployment-target-metrics.component.html',
  imports: [OverlayModule, BytesPipe, PercentPipe, NgStyle, FaIconComponent, StatusDotDirective],
  animations: [modalFlyInOut, drawerFlyInOut, dropdownAnimation],
  styleUrls: ['./deployment-target-metrics.component.scss'],
})
export class DeploymentTargetMetricsComponent {
  public readonly metrics = input.required<DeploymentTargetLatestMetrics>();
  protected readonly hovered = signal(false);
  protected readonly anyDiskWarning = computed(() =>
    this.metrics().diskMetrics?.some((disk) => disk.bytesUsed / disk.bytesTotal > 0.75)
  );

  protected readonly faHardDrive = faHardDrive;
  protected readonly faExclamation = faExclamation;

  protected getUsageDegrees(value: number | undefined): string {
    return (360 * (value ?? 0)).toFixed() + 'deg';
  }
}
