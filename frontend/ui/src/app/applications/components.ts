import {AsyncPipe, NgOptimizedImage} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, input} from '@angular/core';
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

/** An application with its logo, deployment type and, where one is meant, a version. */
@Component({
  selector: 'app-application-preview',
  template: `
    <app-application-logo class="size-10 shrink-0" [imageUrl]="imageUrl()" [type]="type()" />
    <div class="min-w-0">
      <div class="truncate" [title]="label()">{{ label() }}</div>
      <span class="distr-deployment-type-badge capitalize">{{ type() }}</span>
    </div>
  `,
  host: {class: 'flex items-center gap-2'},
  imports: [ApplicationLogoComponent],
})
export class ApplicationPreviewComponent {
  public readonly name = input.required<string>();
  public readonly type = input.required<DeploymentType>();
  public readonly imageUrl = input<string>();
  public readonly version = input<string>();

  protected readonly label = computed(() => [this.name(), this.version()].filter((part) => part).join(' '));
}
