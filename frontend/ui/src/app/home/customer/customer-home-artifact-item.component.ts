import {DatePipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {RelativeDatePipe} from '../../../util/dates';
import {CustomerHomeArtifact} from '../../types/customer-home';
import {CustomerHomeTitleImageComponent} from './customer-home-title-image.component';

@Component({
  selector: 'app-customer-home-artifact-item',
  templateUrl: './customer-home-artifact-item.component.html',
  imports: [DatePipe, RouterLink, RelativeDatePipe, CustomerHomeTitleImageComponent],
  host: {
    class:
      'relative z-10 flex flex-col sm:flex-row gap-3 px-2 py-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700',
  },
})
export class CustomerHomeArtifactItemComponent {
  public readonly artifact = input.required<CustomerHomeArtifact>();

  protected readonly pulledBy = computed(() => {
    const pull = this.artifact().lastPull;
    return pull?.pulledBy?.name ?? pull?.deploymentTargetName;
  });
}
