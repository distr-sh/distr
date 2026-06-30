import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, input} from '@angular/core';
import {SecureImagePipe} from '../../util/secureImage';

@Component({
  selector: 'app-user-avatar',
  changeDetection: ChangeDetectionStrategy.Eager,
  host: {class: 'inline-block shrink-0'},
  imports: [AsyncPipe, SecureImagePipe],
  template: `
    @if (image()) {
      <img [attr.src]="image()! | secureImage | async" [alt]="name()" class="size-full rounded-full object-cover" />
    } @else {
      <div
        class="size-full rounded-full bg-primary-100 dark:bg-primary-900 text-primary-700 dark:text-primary-300 flex items-center justify-center font-medium"
        [class]="initialsClass()">
        {{ initials() ?? '' }}
      </div>
    }
  `,
})
export class UserAvatarComponent {
  /** Image UUID or URL. The secureImage pipe resolves a bare UUID to the files endpoint. */
  public readonly image = input<string>();
  public readonly name = input<string | undefined>();
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
