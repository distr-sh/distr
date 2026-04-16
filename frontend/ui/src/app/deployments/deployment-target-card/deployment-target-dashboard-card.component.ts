import {OverlayModule} from '@angular/cdk/overlay';
import {TextFieldModule} from '@angular/cdk/text-field';
import {AsyncPipe, NgOptimizedImage} from '@angular/common';
import {Component, input} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {DeploymentTarget} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowUpRightFromSquare} from '@fortawesome/free-solid-svg-icons';
import {SecureImagePipe} from '../../../util/secureImage';
import {DeploymentTargetLatestMetrics} from '../../types/deployment-target-metrics';
import {DeploymentAppNameComponent} from './deployment-app-name.component';
import {DeploymentStatusTextComponent} from './deployment-status-text.component';
import {DeploymentTargetMetricsComponent} from './deployment-target-metrics.component';

@Component({
  selector: 'app-deployment-target-dashboard-card',
  templateUrl: './deployment-target-dashboard-card.component.html',
  imports: [
    NgOptimizedImage,
    FaIconComponent,
    OverlayModule,
    ReactiveFormsModule,
    DeploymentTargetMetricsComponent,
    TextFieldModule,
    DeploymentAppNameComponent,
    DeploymentStatusTextComponent,
    SecureImagePipe,
    AsyncPipe,
    RouterLink,
  ],
})
export class DeploymentTargetDashboardCardComponent {
  public readonly deploymentTarget = input.required<DeploymentTarget>();
  public readonly deploymentTargetMetrics = input<DeploymentTargetLatestMetrics | undefined>(undefined);

  protected readonly faArrowUpRightFromSquare = faArrowUpRightFromSquare;
}
