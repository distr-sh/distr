import {DatePipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowUpRightFromSquare, faServer} from '@fortawesome/free-solid-svg-icons';
import {RelativeDatePipe} from '../../../util/dates';
import {DeploymentStatusTextComponent} from '../../deployments/deployment-target-card/deployment-status-text.component';
import {CustomerHomeDeploymentTarget} from '../../types/customer-home';
import {CustomerHomeTitleImageComponent} from './customer-home-title-image.component';

@Component({
  selector: 'app-customer-home-deployment-target-item',
  templateUrl: './customer-home-deployment-target-item.component.html',
  imports: [
    DatePipe,
    RouterLink,
    FaIconComponent,
    RelativeDatePipe,
    DeploymentStatusTextComponent,
    CustomerHomeTitleImageComponent,
  ],
  host: {
    class: 'relative z-10 flex flex-col gap-2 px-2 py-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700',
    '[class.xl:col-span-2]': 'hasMultipleDeployments()',
  },
})
export class CustomerHomeDeploymentTargetItemComponent {
  public readonly deploymentTarget = input.required<CustomerHomeDeploymentTarget>();

  protected readonly faArrowUpRightFromSquare = faArrowUpRightFromSquare;
  protected readonly faServer = faServer;
  protected readonly hasMultipleDeployments = computed(() => this.deploymentTarget().deployments.length > 1);
}
