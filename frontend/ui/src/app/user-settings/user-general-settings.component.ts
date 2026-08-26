import {AsyncPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, signal} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPen, faUser} from '@fortawesome/free-solid-svg-icons';
import {filter, firstValueFrom, take} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {SecureImagePipe} from '../../util/secureImage';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {ContextService} from '../services/context.service';
import {ImageUploadService} from '../services/image-upload.service';
import {SettingsService} from '../services/settings.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-user-general-settings',
  templateUrl: './user-general-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, SecureImagePipe, AsyncPipe],
})
export class UserGeneralSettingsComponent {
  protected readonly faPen = faPen;
  protected readonly faUser = faUser;

  private readonly fb = inject(FormBuilder);
  private readonly ctx = inject(ContextService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly settingsService = inject(SettingsService);

  protected readonly user = toSignal(this.ctx.getUser());

  protected readonly customOidcSession = this.auth.isCustomOidcSession();

  protected readonly generalForm = this.fb.group({
    name: this.fb.control(''),
    imageId: this.fb.control(''),
  });

  protected readonly emailForm = this.fb.group({
    email: this.fb.control('', [Validators.required, Validators.email]),
  });

  protected readonly formLoading = signal(true);

  constructor() {
    this.ctx
      .getUser()
      .pipe(take(1), takeUntilDestroyed())
      .subscribe((user) => {
        this.generalForm.patchValue(user);
        this.formLoading.set(false);
      });
  }

  protected showProfilePictureDialog() {
    this.imageUploadService
      .showDialog({imageUrl: this.user()?.imageUrl, scope: 'platform', showSuccessNotification: false})
      .pipe(filter((id) => id !== null))
      .subscribe((imageId) => {
        this.generalForm.patchValue({imageId});
        this.generalForm.markAsDirty();
      });
  }

  protected async saveGeneral(): Promise<void> {
    if (this.generalForm.invalid) {
      this.generalForm.markAllAsTouched();
      return;
    }

    try {
      this.formLoading.set(true);
      const result = await firstValueFrom(
        this.settingsService.updateUserSettings({
          name: this.generalForm.value.name ?? undefined,
          imageId: this.generalForm.value.imageId || undefined,
        })
      );
      this.toast.success('User settings saved successfully.');
      this.generalForm.patchValue(result);
      this.generalForm.markAsPristine();
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
  }

  protected async saveEmail(): Promise<void> {
    const email = this.emailForm.value.email;
    if (this.emailForm.invalid || !email) {
      this.emailForm.markAllAsTouched();
      return;
    }

    try {
      this.formLoading.set(true);
      await firstValueFrom(this.settingsService.requestEmailVerification(email));
      this.toast.success('Verification request sent. Please check your inbox.');
      this.emailForm.reset();
    } catch (e) {
      const errorMessage = getFormDisplayedError(e);
      if (errorMessage) {
        this.toast.error(errorMessage);
      }
    } finally {
      this.formLoading.set(false);
    }
  }
}
