import {AsyncPipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, inject, OnInit, signal} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {OrganizationBranding} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk} from '@fortawesome/free-solid-svg-icons';
import {MarkdownPipe} from 'ngx-markdown';
import {lastValueFrom, map, Observable} from 'rxjs';
import {base64ToBlob} from '../../util/blob';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {OrganizationBrandingService} from '../services/organization-branding.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-organization-branding',
  templateUrl: './organization-branding.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, AsyncPipe, AutotrimDirective, MarkdownPipe],
})
export class OrganizationBrandingComponent implements OnInit {
  protected readonly faFloppyDisk = faFloppyDisk;

  protected readonly auth = inject(AuthService);
  private readonly organizationBrandingService = inject(OrganizationBrandingService);
  private readonly toast = inject(ToastService);

  private organizationBranding?: OrganizationBranding;

  protected markdownPreviewMode = false;

  protected readonly form = new FormGroup({
    title: new FormControl(''),
    description: new FormControl(''),
    logo: new FormControl<Blob | null>(null),
  });
  formLoading = signal(false);
  protected readonly logoSrc: Observable<string | null> = this.form.controls.logo.valueChanges.pipe(
    map((logo) => (logo ? URL.createObjectURL(logo) : null))
  );

  async ngOnInit() {
    try {
      this.organizationBranding = await lastValueFrom(this.organizationBrandingService.get());
      this.form.patchValue({
        title: this.organizationBranding.title,
        description: this.organizationBranding.description,
        logo: this.organizationBranding.logo
          ? base64ToBlob(this.organizationBranding.logo, this.organizationBranding.logoContentType)
          : null,
      });
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
        // it's a valid use case for an organization to have no branding (hence 404 is not shown in toast)
        this.toast.error(msg);
      }
    }
  }

  async save() {
    this.form.markAllAsTouched();
    if (this.form.valid) {
      this.formLoading.set(true);
      const formData = new FormData();
      formData.set('title', this.form.value.title ?? '');
      formData.set('description', this.form.value.description ?? '');
      formData.set('logo', this.form.value.logo ? (this.form.value.logo as File) : '');

      const id = this.organizationBranding?.id;
      let req: Observable<OrganizationBranding>;
      if (id) {
        req = this.organizationBrandingService.update(formData);
      } else {
        req = this.organizationBrandingService.create(formData);
      }

      try {
        this.organizationBranding = await lastValueFrom(req);
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

  onLogoChange(event: Event) {
    const file = (event.target as HTMLInputElement).files?.[0];
    this.form.patchValue({logo: file ?? null});
  }

  deleteLogo() {
    this.form.patchValue({logo: null});
  }
}
