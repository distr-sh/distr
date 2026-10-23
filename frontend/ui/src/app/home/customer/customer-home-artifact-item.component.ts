import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBox} from '@fortawesome/free-solid-svg-icons';
import {RelativeDatePipe} from '../../../util/dates';
import {SecureImagePipe} from '../../../util/secureImage';
import {CustomerHomeArtifact} from '../../types/customer-home';

@Component({
  selector: 'app-customer-home-artifact-item',
  templateUrl: './customer-home-artifact-item.component.html',
  imports: [AsyncPipe, DatePipe, RouterLink, FaIconComponent, RelativeDatePipe, SecureImagePipe],
  host: {
    class:
      'group relative flex flex-col sm:flex-row gap-3 px-2 py-1.5 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700',
  },
})
export class CustomerHomeArtifactItemComponent {
  public readonly artifact = input.required<CustomerHomeArtifact>();

  protected readonly faBox = faBox;

  protected readonly titleImage = computed(() => this.artifact().titleImageUrl ?? this.artifact().imageUrl);
  protected readonly pulledBy = computed(() => {
    const pull = this.artifact().lastPull;
    return pull?.pulledBy?.name ?? pull?.deploymentTargetName;
  });
}
