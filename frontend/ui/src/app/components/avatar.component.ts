import {AsyncPipe} from '@angular/common';
import {Component, computed, input} from '@angular/core';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {initials} from '../../util/initials';
import {SecureImagePipe} from '../../util/secureImage';

/**
 * A user's profile image, falling back to their initials.
 *
 * Both renderings fill the host, so the caller must give it a size (`class="size-8"`). Without
 * one the host collapses onto its content, leaving an uploaded image at its natural resolution.
 */
@Component({
  selector: 'app-avatar',
  host: {class: 'inline-block shrink-0'},
  imports: [AsyncPipe, SecureImagePipe, FaIconComponent],
  template: `
    @if (image(); as image) {
      <img
        [attr.src]="image | secureImage | async"
        [alt]="name() ?? ''"
        class="size-full rounded-[inherit]"
        [class.object-contain]="imageIsLogo()" />
    } @else if (icon(); as fallbackIcon) {
      <span
        class="flex size-full items-center justify-center rounded-[inherit] bg-gray-400 text-white dark:text-gray-800">
        <fa-icon [icon]="fallbackIcon" [class]="iconClass()" />
      </span>
    } @else {
      <div
        class="flex size-full items-center justify-center rounded-[inherit] bg-primary-100 font-medium text-primary-700 dark:bg-primary-900 dark:text-primary-300"
        [class]="initialsClass()">
        {{ initials() ?? '' }}
      </div>
    }
  `,
})
export class AvatarComponent {
  /** Image UUID or URL. The secureImage pipe resolves a bare UUID to the files endpoint. */
  public readonly image = input<string>();
  public readonly name = input<string | undefined>();
  /** Fits a non-square image such as an organization logo inside the shape instead of stretching it. */
  public readonly imageIsLogo = input(false);
  /** Fallback icon. Without it the initials of the name are shown. */
  public readonly icon = input<IconDefinition>();
  public readonly iconClass = input<string>('text-lg');
  public readonly initialsClass = input<string>('text-xs');

  protected readonly initials = computed(() => initials(this.name()));
}
