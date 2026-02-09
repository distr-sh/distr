import {OverlayModule} from '@angular/cdk/overlay';
import {TextFieldModule} from '@angular/cdk/text-field';
import {NgOptimizedImage} from '@angular/common';
import {Component} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {IsStalePipe} from '../../../util/model';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {ConnectInstructionsComponent} from '../../components/connect-instructions/connect-instructions.component';
import {DeploymentStatusDotDirective, StatusDotComponent} from '../../components/status-dot';
import {DeploymentModalComponent} from '../deployment-modal.component';
import {DeploymentTargetCardBaseComponent} from './deployment-target-card-base.component';
import {DeploymentTargetMetricsComponent} from './deployment-target-metrics.component';

@Component({
  selector: 'app-deployment-target-dashboard-card',
  templateUrl: './deployment-target-dashboard-card.component.html',
  imports: [
    NgOptimizedImage,
    StatusDotComponent,
    DeploymentStatusDotDirective,
    FaIconComponent,
    OverlayModule,
    ConnectInstructionsComponent,
    ReactiveFormsModule,
    DeploymentModalComponent,
    DeploymentTargetMetricsComponent,
    TextFieldModule,
    IsStalePipe,
  ],
  animations: [modalFlyInOut, drawerFlyInOut, dropdownAnimation],
})
export class DeploymentTargetDashboardCardComponent extends DeploymentTargetCardBaseComponent {}
