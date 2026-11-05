import {inject, Injectable} from '@angular/core';
import {ImageUploadContext, ImageUploadDialogComponent} from '../components/image-upload/image-upload-dialog.component';
import {OverlayService} from './overlay.service';

@Injectable()
export class ImageUploadService {
  private readonly overlay = inject(OverlayService);

  public showDialog(context: ImageUploadContext = {}) {
    return this.overlay.showModal<string>(ImageUploadDialogComponent, {data: context}).result();
  }
}
