import {OverlayModule} from '@angular/cdk/overlay';
import {PercentPipe} from '@angular/common';
import {Component, input, signal} from '@angular/core';
import {BytesPipe} from '../../../util/units';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {DeploymentTargetLatestMetrics} from '../../services/deployment-target-metrics.service';

@Component({
  selector: 'app-deployment-target-metrics',
  templateUrl: './deployment-target-metrics.component.html',
  imports: [OverlayModule, BytesPipe, PercentPipe],
  animations: [modalFlyInOut, drawerFlyInOut, dropdownAnimation],
  styleUrls: ['./deployment-target-metrics.component.scss'],
})
export class DeploymentTargetMetricsComponent {
  public readonly metrics = input.required<DeploymentTargetLatestMetrics>();
  protected readonly hovered = signal(false);

  protected getPercentClass(usage: number | undefined): string {
    const val = Math.ceil((usage || 0) * 100);
    const mod5 = val % 5;
    return (mod5 === 0 ? val : val - mod5 + 5).toFixed();
  }
}
