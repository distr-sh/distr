import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, input, model} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPen, faTrashCan} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {SecureImagePipe} from '../../../util/secureImage';
import {ImageUploadService} from '../../services/image-upload.service';

@Component({
  selector: 'app-branding-image-picker',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [AsyncPipe, FaIconComponent, SecureImagePipe],
  templateUrl: './branding-image-picker.component.html',
})
export class BrandingImagePickerComponent {
  private readonly imageUploadService = inject(ImageUploadService);

  protected readonly faPen = faPen;
  protected readonly faTrashCan = faTrashCan;

  readonly imageId = model<string | undefined>(undefined);
  readonly label = input.required<string>();
  readonly alt = input.required<string>();
  readonly fallbackSrc = input.required<string>();
  readonly accept = input<string>();
  readonly acceptDescription = input<string>();
  readonly imageClass = input('');
  /** Extra classes for the fallback image, if it needs different sizing than the uploaded one. */
  readonly fallbackClass = input<string>();
  /** Resolve the image through the unauthenticated public file endpoint instead of the `secureImage` pipe. */
  readonly publicUrl = input(false);

  protected readonly publicImageUrl = computed(() => {
    const id = this.imageId();
    return id ? `/api/public/v1/files/${id}` : undefined;
  });
  protected readonly currentImageClass = computed(() =>
    this.imageId() ? this.imageClass() : (this.fallbackClass() ?? this.imageClass())
  );

  protected async edit() {
    const fileId = await firstValueFrom(
      this.imageUploadService.showDialog({
        scope: 'organization',
        public: true,
        showSuccessNotification: false,
        accept: this.accept(),
        acceptDescription: this.acceptDescription(),
      })
    );
    if (!fileId || this.imageId() === fileId) {
      return;
    }
    // Stage the uploaded file: it is only attached to the branding when the form is saved.
    this.imageId.set(fileId);
  }

  protected remove() {
    this.imageId.set(undefined);
  }
}
