import {OverlayModule} from '@angular/cdk/overlay';
import {TextFieldModule} from '@angular/cdk/text-field';
import {AsyncPipe, NgOptimizedImage} from '@angular/common';
import {Component} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {SecureImagePipe} from '../../../util/secureImage';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {ConnectInstructionsComponent} from '../../components/connect-instructions/connect-instructions.component';
import {StatusDotComponent} from '../../components/status-dot';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {DeploymentModalComponent} from '../deployment-modal.component';
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
    ConnectInstructionsComponent,
    ReactiveFormsModule,
    DeploymentModalComponent,
    DeploymentTargetMetricsComponent,
    TextFieldModule,
    DeploymentAppNameComponent,
    DeploymentStatusTextComponent,
    AutotrimDirective,
    SecureImagePipe,
    AsyncPipe,
  ],
  animations: [modalFlyInOut, drawerFlyInOut, dropdownAnimation],
})
export class DeploymentTargetDashboardCardComponent extends DeploymentTargetCardBaseComponent {}
