import {Component, input} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faShip} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-deployment-app-name',
  imports: [FaIconComponent],
  template: `
    <div class="flex items-center gap-3">
      <div class="flex-shrink-0">
        <fa-icon class="text-gray-500 dark:text-gray-400 text-2xl" [icon]="faShip"></fa-icon>
      </div>
      <div class="flex-1 min-w-0">
        <div class="font-medium text-gray-900 dark:text-white truncate break-all" [title]="applicationName()">
          {{ applicationName() }}
        </div>
        <div class="text-gray-500 dark:text-gray-400 text-xs truncate break-all" [title]="applicationVersionName()">
          {{ applicationVersionName() }}
        </div>
      </div>
    </div>
  `,
})
export class DeploymentAppNameComponent {
  public readonly applicationName = input.required<string>();
  public readonly applicationVersionName = input.required<string>();
  protected readonly faShip = faShip;
}
