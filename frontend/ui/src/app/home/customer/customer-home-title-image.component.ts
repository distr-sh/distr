import {AsyncPipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {initials} from '../../../util/initials';
import {SecureImagePipe} from '../../../util/secureImage';

@Component({
  selector: 'app-customer-home-title-image',
  templateUrl: './customer-home-title-image.component.html',
  imports: [AsyncPipe, FaIconComponent, SecureImagePipe],
  host: {
    class:
      'sm:w-48 sm:self-start shrink-0 aspect-video rounded-lg overflow-hidden bg-gray-100 dark:bg-gray-700 border-gray-200 dark:border-gray-600 flex items-center justify-center',
    '[class.border]': '!src()',
  },
})
export class CustomerHomeTitleImageComponent {
  public readonly src = input<string>();
  /** Whose initials stand in for a missing image. */
  public readonly name = input<string>();
  /** Stands in for a missing image when there is no name either. */
  public readonly icon = input<IconDefinition>();

  protected readonly initials = computed(() => initials(this.name()));
}
