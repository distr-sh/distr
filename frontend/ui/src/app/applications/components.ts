import {AsyncPipe, NgOptimizedImage} from '@angular/common';
import {ChangeDetectionStrategy, Component, input} from '@angular/core';
import {DeploymentType} from '@distr-sh/distr-sdk';
import {SecureImagePipe} from '../../util/secureImage';

/**
 * The logo of an application, falling back to the generic icon of its deployment type.
 * The host element must provide the sizing.
 */
@Component({
  selector: 'app-application-logo',
  template: `
    @if (imageUrl()) {
      <img class="size-full rounded-sm object-contain" [attr.src]="imageUrl()! | secureImage | async" alt="" />
    } @else {
      <img
        class="size-full rounded-sm object-contain"
        [ngSrc]="'/' + type() + '.png'"
        [alt]="type()"
        height="199"
        width="199" />
    }
  `,
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [AsyncPipe, NgOptimizedImage, SecureImagePipe],
})
export class ApplicationLogoComponent {
  public readonly imageUrl = input<string>();
  public readonly type = input.required<DeploymentType>();
}
