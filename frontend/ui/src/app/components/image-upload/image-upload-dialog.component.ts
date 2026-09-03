import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, OnDestroy, signal} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faXmark} from '@fortawesome/free-solid-svg-icons';
import {lastValueFrom, map, Observable, Subject, takeUntil} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {FileScope, FilesService} from '../../services/files.service';
import {OverlayData} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {ClosableDialog} from '../confirm-dialog/closable-dialog';

const DEFAULT_ACCEPT = 'image/svg+xml,image/png,image/jpeg,image/gif';
const DEFAULT_ACCEPT_DESCRIPTION = 'SVG, PNG, JPG or GIF (recommended size 64px x 64px - square)';

export interface ImageUploadContext {
  scope?: FileScope;
  imageUrl?: string;
  showSuccessNotification?: boolean;
  public?: boolean;
  /** Overrides the accepted file types (input `accept` attribute). Defaults to image types only. */
  accept?: string;
  /** Overrides the help text describing the accepted file types. */
  acceptDescription?: string;
}

@Component({
  imports: [FaIconComponent, ReactiveFormsModule, AsyncPipe],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './image-upload-dialog.component.html',
})
export class ImageUploadDialogComponent extends ClosableDialog<string> implements OnDestroy {
  public readonly faXmark = faXmark;
  public readonly data = inject(OverlayData) as ImageUploadContext;
  protected readonly accept = this.data.accept ?? DEFAULT_ACCEPT;
  protected readonly acceptDescription = this.data.acceptDescription ?? DEFAULT_ACCEPT_DESCRIPTION;
  private readonly destroyed$ = new Subject<void>();
  private toast = inject(ToastService);

  private files = inject(FilesService);
  protected readonly form = new FormGroup({
    image: new FormControl<Blob | null>(null, Validators.required),
  });

  formLoading = signal(false);

  protected readonly imageSrc: Observable<string | null> = this.form.controls.image.valueChanges.pipe(
    takeUntil(this.destroyed$),
    map((image) => (image ? URL.createObjectURL(image) : null))
  );

  onImageChange(event: Event) {
    const file = (event.target as HTMLInputElement).files?.[0];
    this.form.patchValue({image: file ?? null});
  }

  deleteImage() {
    this.form.patchValue({image: null});
  }

  ngOnDestroy(): void {
    this.destroyed$.next();
    this.destroyed$.complete();
  }

  async save() {
    this.form.markAllAsTouched();
    if (this.form.valid) {
      this.formLoading.set(true);
      const formData = new FormData();
      formData.set('file', this.form.value.image as File);
      if (this.data.public) {
        formData.set('public', 'true');
      }

      try {
        let uploadResult = this.files.uploadFile(formData, this.data.scope);
        await this.dialogRef.close(await lastValueFrom(uploadResult));
        if (this.data.showSuccessNotification !== false) {
          this.toast.success('Image saved successfully');
        }
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.formLoading.set(false);
      }
    }
  }
}
