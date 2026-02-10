import {AsyncPipe} from '@angular/common';
import {Component, input} from '@angular/core';
import {Application} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faShip} from '@fortawesome/free-solid-svg-icons';
import {SecureImagePipe} from '../../../util/secureImage';

@Component({
  selector: 'app-deployment-app-name',
  imports: [FaIconComponent, SecureImagePipe, AsyncPipe],
  template: `
    <div class="flex items-center gap-3">
      <div class="flex-shrink-0">
        @if (application().imageId; as imageId) {
          <img class="size-8 rounded-sm" [attr.src]="imageId | secureImage | async" alt="" />
        } @else {
          <fa-icon class="text-gray-500 dark:text-gray-400 text-2xl" [icon]="faShip" />
        }
      </div>
      <div class="flex-1 min-w-0">
        <div class="font-medium text-gray-900 dark:text-white truncate break-all" [title]="application().name">
          {{ application().name }}
        </div>
        <div class="text-gray-500 dark:text-gray-400 text-xs truncate break-all" [title]="applicationVersionName()">
          {{ applicationVersionName() }}
        </div>
      </div>
    </div>
  `,
})
export class DeploymentAppNameComponent {
  public readonly application = input.required<Application>();
  public readonly applicationVersionName = input.required<string>();

  protected readonly faShip = faShip;
}
