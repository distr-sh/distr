import {AsyncPipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {ChangeDetectionStrategy, Component, computed, inject, OnInit, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {OrganizationBranding} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faCircleXmark, faFloppyDisk, faPen, faTrashCan} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, lastValueFrom, map, startWith} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {SecureImagePipe} from '../../util/secureImage';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {InnerMarkdownDirective} from '../directives/inner-markdown.directive';
import {AuthService} from '../services/auth.service';
import {ImageUploadService} from '../services/image-upload.service';
import {OrganizationBrandingService} from '../services/organization-branding.service';
import {PortalBrandingService} from '../services/portal-branding.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-organization-branding',
  templateUrl: './organization-branding.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    FaIconComponent,
    ReactiveFormsModule,
    AsyncPipe,
    AutotrimDirective,
    InnerMarkdownDirective,
    SecureImagePipe,
  ],
})
export class OrganizationBrandingComponent implements OnInit {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faPen = faPen;
  protected readonly faTrashCan = faTrashCan;
  protected readonly faCheck = faCheck;
  protected readonly faCircleXmark = faCircleXmark;

  protected readonly auth = inject(AuthService);
  private readonly organizationBrandingService = inject(OrganizationBrandingService);
  private readonly imageUploadService = inject(ImageUploadService);
  private readonly portalBranding = inject(PortalBrandingService);
  private readonly toast = inject(ToastService);

  private organizationBranding?: OrganizationBranding;

  protected markdownPreviewMode = false;

  protected readonly logoImageId = signal<string | undefined>(undefined);
  protected readonly faviconImageId = signal<string | undefined>(undefined);
  protected readonly faviconImageUrl = computed(() => {
    const id = this.faviconImageId();
    return id ? `/api/public/v1/files/${id}` : undefined;
  });
  protected readonly appDomain = signal<string | undefined>(undefined);
  protected readonly registryDomain = signal<string | undefined>(undefined);
  protected readonly emailFromAddress = signal<string | undefined>(undefined);
  protected readonly hasCustomDomains = computed(
    () => !!(this.appDomain() || this.registryDomain() || this.emailFromAddress())
  );
  protected readonly customDomainsData = computed(() => [
    {
      label: 'App domain',
      value: this.appDomain(),
      description: 'Where users and customers access the Distr web application.',
    },
    {
      label: 'Registry domain',
      value: this.registryDomain(),
      description: 'Where users and customers access the Distr artifact registry.',
    },
    {
      label: 'E-mail sender address',
      value: this.emailFromAddress(),
      description: 'The address used to send transactional e-mails to your users and customers.',
    },
  ]);

  protected readonly form = new FormGroup({
    title: new FormControl(''),
    description: new FormControl(''),
    pageTitle: new FormControl(''),
  });
  formLoading = signal(false);
  protected readonly customerPortalName = toSignal(
    this.form.controls.title.valueChanges.pipe(
      startWith(this.form.controls.title.value),
      map((title) => title?.trim() || 'Customer Portal')
    ),
    {initialValue: 'Customer Portal'}
  );

  async ngOnInit() {
    try {
      this.organizationBranding = await lastValueFrom(this.organizationBrandingService.get());
      this.logoImageId.set(this.organizationBranding.logoImageId);
      this.faviconImageId.set(this.organizationBranding.faviconImageId);
      this.appDomain.set(this.organizationBranding.appDomain);
      this.registryDomain.set(this.organizationBranding.registryDomain);
      this.emailFromAddress.set(this.organizationBranding.emailFromAddress);
      this.form.patchValue({
        title: this.organizationBranding.title,
        description: this.organizationBranding.description,
        pageTitle: this.organizationBranding.pageTitle,
      });
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
        // it's a valid use case for an organization to have no branding (hence 404 is not shown in toast)
        this.toast.error(msg);
      }
    }
  }

  async editLogo() {
    const fileId = await firstValueFrom(
      this.imageUploadService.showDialog({scope: 'organization', public: true, showSuccessNotification: false})
    );
    if (!fileId || this.logoImageId() === fileId) {
      return;
    }
    // Stage the uploaded file: it is only attached to the branding when the form is saved.
    this.logoImageId.set(fileId);
  }

  removeLogo() {
    this.logoImageId.set(undefined);
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
    if (!fileId || this.faviconImageId() === fileId) {
      return;
    }
    // Stage the uploaded file: it is only attached to the branding when the form is saved.
    this.faviconImageId.set(fileId);
  }

  removeFavicon() {
    this.faviconImageId.set(undefined);
  }

  async save() {
    this.form.markAllAsTouched();
    if (this.form.valid) {
      this.formLoading.set(true);
      const payload: OrganizationBranding = {
        title: this.form.value.title ?? undefined,
        description: this.form.value.description ?? undefined,
        logoImageId: this.logoImageId(),
        pageTitle: this.form.value.pageTitle?.trim() || undefined,
        faviconImageId: this.faviconImageId(),
      };

      try {
        this.organizationBranding = await lastValueFrom(this.organizationBrandingService.upsert(payload));
        this.logoImageId.set(this.organizationBranding.logoImageId);
        this.faviconImageId.set(this.organizationBranding.faviconImageId);
        // Reflect the saved page title and favicon in the browser tab immediately, without a reload.
        this.portalBranding.apply();
        this.toast.success('Branding saved successfully');
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
