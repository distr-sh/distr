import {OverlayModule} from '@angular/cdk/overlay';
import {TextFieldModule} from '@angular/cdk/text-field';
import {AsyncPipe, NgOptimizedImage} from '@angular/common';
import {Component} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {SecureImagePipe} from '../../../util/secureImage';
import {StatusDotComponent} from '../../components/status-dot';
import {DeploymentAppNameComponent} from './deployment-app-name.component';
import {DeploymentStatusTextComponent} from './deployment-status-text.component';
import {DeploymentTargetCardBaseComponent} from './deployment-target-card-base.component';
import {DeploymentTargetMetricsComponent} from './deployment-target-metrics.component';

@Component({
  selector: 'app-deployment-target-dashboard-card',
  templateUrl: './deployment-target-dashboard-card.component.html',
  imports: [
    NgOptimizedImage,
    StatusDotComponent,
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
export class DeploymentTargetDashboardCardComponent extends DeploymentTargetCardBaseComponent {}
