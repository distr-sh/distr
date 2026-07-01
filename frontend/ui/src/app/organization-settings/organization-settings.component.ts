import {
  ChangeDetectionStrategy,
  Component,
  computed,
  inject,
  OnInit,
  signal,
  TemplateRef,
  viewChild,
} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk, faLightbulb, faPen, faTrashCan} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {slugMaxLength, slugPattern} from '../../util/slug';
import {DeleteOrganizationComponent} from '../components/delete-organization/delete-organization.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {ImageUploadService} from '../services/image-upload.service';
import {OrganizationService} from '../services/organization.service';
import {OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {Organization} from '../types/organization';

@Component({
  selector: 'app-organization-settings',
  templateUrl: './organization-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective, DeleteOrganizationComponent],
})
export class OrganizationSettingsComponent implements OnInit {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faLightbulb = faLightbulb;
  protected readonly faPen = faPen;
  protected readonly faTrashCan = faTrashCan;

  private readonly organizationService = inject(OrganizationService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly overlayService = inject(OverlayService);
  protected readonly auth = inject(AuthService);

  private readonly preflightConfirmTemplate = viewChild.required<TemplateRef<unknown>>('preflightConfirmTemplate');

  private organization?: Organization;

  protected readonly form = this.fb.group({
    name: this.fb.control('', [Validators.required]),
    slug: this.fb.control('', [Validators.pattern(slugPattern), Validators.maxLength(slugMaxLength)]),
    pageTitle: this.fb.control<string | undefined>(undefined),
    faviconImageId: this.fb.control<string | undefined>(undefined),
    appDomain: this.fb.control<string | undefined>({value: undefined, disabled: true}),
    registryDomain: this.fb.control<string | undefined>({value: undefined, disabled: true}),
    emailFromAddress: this.fb.control<string | undefined>({value: undefined, disabled: true}),
    preConnectScript: this.fb.control<string | undefined>(undefined),
    postConnectScript: this.fb.control<string | undefined>(undefined),
    connectScriptIsSudo: this.fb.control<boolean>(false),
    artifactVersionMutable: this.fb.control<boolean>(false),
    prePostScriptsEnabled: this.fb.control<boolean>(false),
  });
  formLoading = signal(false);

  protected readonly isPrePostScriptEnabled = toSignal(
    this.form.controls.prePostScriptsEnabled.valueChanges.pipe(
      startWith(this.form.controls.prePostScriptsEnabled.value)
    ),
    {initialValue: false}
  );

  private readonly faviconImageId = toSignal(
    this.form.controls.faviconImageId.valueChanges.pipe(startWith(this.form.controls.faviconImageId.value)),
    {initialValue: this.form.controls.faviconImageId.value}
  );
  protected readonly faviconImageUrl = computed(() => {
    const id = this.faviconImageId();
    return id ? `/api/public/v1/files/${id}` : undefined;
  });

  async ngOnInit() {
    try {
      this.organization = await firstValueFrom(this.organizationService.get());
      if (this.organization.slug) {
        this.form.controls.slug.addValidators([Validators.required]);
      }
      this.form.patchValue({
        ...this.organization,
        artifactVersionMutable: this.organization.features?.includes('artifact_version_mutable') ?? false,
        prePostScriptsEnabled: this.organization.features?.includes('pre_post_scripts') ?? false,
      });
      if (!this.organization.appDomain) {
        this.form.controls.pageTitle.disable();
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  async save() {
    this.form.markAllAsTouched();
    if (this.form.valid) {
      const wasPrePostScriptsEnabled = this.organization?.features?.includes('pre_post_scripts') ?? false;
      const isNowPrePostScriptsEnabled = this.form.value.prePostScriptsEnabled ?? false;

      if (!wasPrePostScriptsEnabled && isNowPrePostScriptsEnabled) {
        const confirmed = await firstValueFrom(
          this.overlayService.confirm({
            customTemplate: this.preflightConfirmTemplate(),
            message: {
              message: '',
              alert: {
                type: 'warning',
                message: 'Existing agents are not affected. New agent connect commands will use the new format.',
              },
            },
          })
        );
        if (!confirmed) {
          return;
        }
      }

      this.formLoading.set(true);
      try {
        this.organization = await firstValueFrom(
          this.organizationService.update({
            ...this.organization!,
            name: this.form.value.name?.trim()!,
            slug: this.form.value.slug?.trim(),
            pageTitle: this.form.value.pageTitle?.trim() || undefined,
            faviconImageId: this.form.controls.faviconImageId.value,
            preConnectScript: this.form.value.preConnectScript?.trim(),
            postConnectScript: this.form.value.postConnectScript?.trim(),
            connectScriptIsSudo: this.form.value.connectScriptIsSudo ?? false,
            artifactVersionMutable: this.form.value.artifactVersionMutable ?? false,
            prePostScriptsEnabled: this.form.value.prePostScriptsEnabled ?? false,
          })
        );
        this.toast.success('Settings saved successfully');
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

  async editFavicon() {
    const fileId = await firstValueFrom(
      this.imageUploadService.showDialog({
        scope: 'organization',
        public: true,
        showSuccessNotification: false,
        imageUrl: this.faviconImageUrl(),
        accept: 'image/svg+xml,image/png,image/gif,image/x-icon,image/vnd.microsoft.icon,.ico',
        acceptDescription: 'SVG, PNG, GIF or ICO (recommended size 64px x 64px - square)',
      })
    );
    if (!fileId || this.faviconImageUrl()?.includes(fileId)) {
      return;
    }
    // Stage the uploaded file: it is only attached to the organization when the form is saved.
    this.form.controls.faviconImageId.setValue(fileId);
    this.form.markAsDirty();
  }

  removeFavicon() {
    this.form.controls.faviconImageId.setValue(undefined);
    this.form.markAsDirty();
  }
}
