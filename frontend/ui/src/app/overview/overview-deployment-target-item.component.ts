import {AsyncPipe, DatePipe, NgTemplateOutlet} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowUpRightFromSquare, faServer} from '@fortawesome/free-solid-svg-icons';
import {RelativeDatePipe} from '../../util/dates';
import {SecureImagePipe} from '../../util/secureImage';
import {DeploymentStatusTextComponent} from '../deployments/deployment-target-card/deployment-status-text.component';
import {OverviewDeploymentTarget} from '../types/overview';

@Component({
  selector: 'app-overview-deployment-target-item',
  templateUrl: './overview-deployment-target-item.component.html',
  imports: [
    AsyncPipe,
    DatePipe,
    NgTemplateOutlet,
    RouterLink,
    FaIconComponent,
    RelativeDatePipe,
    SecureImagePipe,
    DeploymentStatusTextComponent,
  ],
  host: {
    class: 'group relative flex flex-col gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700',
    '[class.xl:col-span-2]': 'hasMultipleDeployments()',
  },
})
export class OverviewDeploymentTargetItemComponent {
  public readonly deploymentTarget = input.required<OverviewDeploymentTarget>();

  protected readonly faArrowUpRightFromSquare = faArrowUpRightFromSquare;
  protected readonly faServer = faServer;

  protected readonly hasMultipleDeployments = computed(() => this.deploymentTarget().deployments.length > 1);
}
