import {OverlayModule} from '@angular/cdk/overlay';
import {TextFieldModule} from '@angular/cdk/text-field';
import {DatePipe, NgOptimizedImage, NgTemplateOutlet} from '@angular/common';
import {Component} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {drawerFlyInOut} from '../../animations/drawer';
import {dropdownAnimation} from '../../animations/dropdown';
import {modalFlyInOut} from '../../animations/modal';
import {ConnectInstructionsComponent} from '../../components/connect-instructions/connect-instructions.component';
import {StatusDotComponent} from '../../components/status-dot';
import {UuidComponent} from '../../components/uuid';
import {DeploymentModalComponent} from '../deployment-modal.component';
import {DeploymentStatusModalComponent} from '../deployment-status-modal/deployment-status-modal.component';
import {DeploymentTargetStatusModalComponent} from '../deployment-target-status-modal/deployment-target-status-modal.component';
import {DeploymentAppNameComponent} from './deployment-app-name.component';
import {DeploymentStatusTextComponent} from './deployment-status-text.component';
import {DeploymentTargetCardBaseComponent} from './deployment-target-card-base.component';
import {DeploymentTargetMetricsComponent} from './deployment-target-metrics.component';

@Component({
  selector: 'app-deployment-target-card',
  templateUrl: './deployment-target-card.component.html',
  imports: [
    NgOptimizedImage,
    StatusDotComponent,
    UuidComponent,
    DatePipe,
    FaIconComponent,
    OverlayModule,
    ConnectInstructionsComponent,
    ReactiveFormsModule,
    DeploymentModalComponent,
    DeploymentTargetMetricsComponent,
    NgTemplateOutlet,
    DeploymentStatusModalComponent,
    TextFieldModule,
    DeploymentTargetStatusModalComponent,
    DeploymentAppNameComponent,
    DeploymentStatusTextComponent,
  ],
  animations: [modalFlyInOut, drawerFlyInOut, dropdownAnimation],
})
export class DeploymentTargetCardComponent extends DeploymentTargetCardBaseComponent {}
