import {AsyncPipe} from '@angular/common';
import {Component, inject, input, output} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faImage} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {SecureImagePipe} from '../../../util/secureImage';
import {AuthService} from '../../services/auth.service';
import {ImageUploadService} from '../../services/image-upload.service';

const TITLE_IMAGE_ACCEPT_DESCRIPTION = 'SVG, PNG, JPG or GIF (recommended size 960px x 540px, max 500 KiB)';

@Component({
  selector: 'app-title-image-picker',
  imports: [AsyncPipe, SecureImagePipe, FaIconComponent],
  templateUrl: './title-image-picker.component.html',
  host: {class: 'flex items-center gap-3'},
})
export class TitleImagePickerComponent {
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly auth = inject(AuthService);

  public readonly imageUrl = input<string>();
  public readonly picked = output<string>();

  protected readonly editable = this.auth.hasAnyRole('read_write', 'admin');

  protected readonly faImage = faImage;

  protected async pick() {
    const imageUrl = this.imageUrl();
    const fileId = await firstValueFrom(
      this.imageUploadService.showDialog({imageUrl, acceptDescription: TITLE_IMAGE_ACCEPT_DESCRIPTION})
    );
    if (fileId && !imageUrl?.includes(fileId)) {
      this.picked.emit(fileId);
    }
  }
}
