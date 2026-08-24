import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, input} from '@angular/core';
import {FaIconComponent, IconDefinition} from '@fortawesome/angular-fontawesome';
import {SecureImagePipe} from '../../util/secureImage';

@Component({
  selector: 'app-avatar',
  changeDetection: ChangeDetectionStrategy.Eager,
  host: {class: 'inline-block shrink-0'},
  imports: [AsyncPipe, SecureImagePipe, FaIconComponent],
  template: `
    @if (image()) {
      <img [attr.src]="image()! | secureImage | async" [alt]="name() ?? ''" class="size-full rounded-[inherit]" />
    } @else if (icon(); as fallbackIcon) {
      <span
        class="size-full rounded-[inherit] bg-gray-400 text-white dark:text-gray-800 flex items-center justify-center">
        <fa-icon [icon]="fallbackIcon" [class]="iconClass()" />
      </span>
    } @else {
      <div
        class="size-full rounded-[inherit] bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300 flex items-center justify-center font-medium"
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
  /** Fallback icon. Without it the initials of the name are shown. */
  public readonly icon = input<IconDefinition>();
  public readonly iconClass = input<string>('text-lg');
  public readonly initialsClass = input<string>('text-xs');

  protected readonly initials = computed(() =>
    this.name()
      ?.split(' ')
      .map((part) => part.charAt(0))
      .join('')
      .toUpperCase()
      .substring(0, 2)
  );
}
