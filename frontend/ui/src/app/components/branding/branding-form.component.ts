import {AsyncPipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {ChangeDetectionStrategy, Component, computed, inject, input, OnInit, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {OrganizationBranding} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCheck, faCircleXmark} from '@fortawesome/free-solid-svg-icons';
import {lastValueFrom, map, startWith} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {SecureImagePipe} from '../../../util/secureImage';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {InnerMarkdownDirective} from '../../directives/inner-markdown.directive';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {PortalBrandingService} from '../../services/portal-branding.service';
import {ToastService} from '../../services/toast.service';
import {BrandingImagePickerComponent} from './branding-image-picker.component';

@Component({
  selector: 'app-branding-form',
  templateUrl: './branding-form.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    AsyncPipe,
    AutotrimDirective,
    BrandingImagePickerComponent,
    FaIconComponent,
    InnerMarkdownDirective,
    ReactiveFormsModule,
    SecureImagePipe,
  ],
})
export class BrandingFormComponent implements OnInit {
  private readonly organizationBrandingService = inject(OrganizationBrandingService);
  private readonly portalBranding = inject(PortalBrandingService);
  private readonly toast = inject(ToastService);

  protected readonly faCheck = faCheck;
  protected readonly faCircleXmark = faCircleXmark;

  /** Welcome page content prefilled for organizations that have not written one yet. */
  readonly defaultDescription = input('');
  readonly showCustomDomains = input(true);

  protected markdownPreviewMode = false;

  protected readonly logoImageId = signal<string | undefined>(undefined);
  protected readonly faviconImageId = signal<string | undefined>(undefined);
  private readonly savedLogoImageId = signal<string | undefined>(undefined);
  private readonly savedFaviconImageId = signal<string | undefined>(undefined);

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

  readonly form = new FormGroup({
    title: new FormControl(''),
    description: new FormControl(''),
    pageTitle: new FormControl(''),
  });
  protected readonly customerPortalName = toSignal(
    this.form.controls.title.valueChanges.pipe(
      startWith(this.form.controls.title.value),
      map((title) => title?.trim() || 'Customer Portal')
    ),
    {initialValue: 'Customer Portal'}
  );

  async ngOnInit() {
    try {
      this.applyBranding(await lastValueFrom(this.organizationBrandingService.get()));
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
        // it's a valid use case for an organization to have no branding (hence 404 is not shown in toast)
        this.toast.error(msg);
      }
    }

    const defaultDescription = this.defaultDescription();
    if (defaultDescription && !this.form.value.description) {
      this.form.controls.description.setValue(defaultDescription);
      // Mark dirty so the prefilled default is persisted instead of being silently discarded.
      this.form.controls.description.markAsDirty();
    }
  }

  /** Whether the form or one of the staged images differs from the last saved branding. */
  dirty(): boolean {
    return (
      this.form.dirty ||
      this.logoImageId() !== this.savedLogoImageId() ||
      this.faviconImageId() !== this.savedFaviconImageId()
    );
  }

  async save(): Promise<boolean> {
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      return false;
    }

    const payload: OrganizationBranding = {
      title: this.form.value.title?.trim() || undefined,
      description: this.form.value.description ?? undefined,
      logoImageId: this.logoImageId(),
      pageTitle: this.form.value.pageTitle?.trim() || undefined,
      faviconImageId: this.faviconImageId(),
    };

    try {
      this.applyBranding(await lastValueFrom(this.organizationBrandingService.upsert(payload)));
      this.form.markAsPristine();
      // Reflect the saved page title and favicon in the browser tab immediately, without a reload.
      this.portalBranding.apply();
      this.toast.success('Branding saved successfully');
      return true;
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
      return false;
    }
  }

  private applyBranding(branding: OrganizationBranding) {
    this.logoImageId.set(branding.logoImageId);
    this.savedLogoImageId.set(branding.logoImageId);
    this.faviconImageId.set(branding.faviconImageId);
    this.savedFaviconImageId.set(branding.faviconImageId);
    this.appDomain.set(branding.appDomain);
    this.registryDomain.set(branding.registryDomain);
    this.emailFromAddress.set(branding.emailFromAddress);
    this.form.patchValue({
      title: branding.title,
      description: branding.description,
      pageTitle: branding.pageTitle,
    });
  }
}
