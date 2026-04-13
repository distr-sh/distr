import {Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGear, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {catchError, EMPTY, filter, firstValueFrom, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AuthService} from '../services/auth.service';
import {LicenseTemplatesService} from '../services/license-templates.service';
import {OrganizationService} from '../services/organization.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {LicenseTemplate} from '../types/license-template';

@Component({
  selector: 'app-billing',
  templateUrl: './billing.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, RouterLink],
})
export class BillingComponent {
  protected readonly auth = inject(AuthService);
  private readonly templatesService = inject(LicenseTemplatesService);
  private readonly organizationService = inject(OrganizationService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faGear = faGear;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;

  protected readonly templates = toSignal(this.templatesService.list(), {initialValue: [] as LicenseTemplate[]});
  protected readonly organization = toSignal(this.organizationService.get());

  protected readonly selectedTemplate = signal<LicenseTemplate | undefined>(undefined);
  protected readonly templateFormLoading = signal(false);

  private drawerRef?: DialogRef;
  private readonly templateDrawerTemplate = viewChild.required<TemplateRef<unknown>>('templateDrawer');

  protected readonly templateForm = this.fb.group({
    name: this.fb.control('', Validators.required),
    payloadTemplate: this.fb.control('', Validators.required),
    expirationGracePeriodDays: this.fb.control(0, [Validators.required, Validators.min(0)]),
  });

  protected readonly payloadTemplatePlaceholder = `{ "plan": "{{ if hasItem \\"pro-monthly\\" }}pro{{ else }}starter{{ end }}", "seats": {{ itemQuantity \\"seats-monthly\\" }} }`;

  openTemplateDrawer(template?: LicenseTemplate) {
    this.hideDrawer();
    this.selectedTemplate.set(template);
    if (template) {
      this.templateForm.patchValue({
        name: template.name,
        payloadTemplate: template.payloadTemplate,
        expirationGracePeriodDays: template.expirationGracePeriodDays,
      });
    } else {
      this.templateForm.reset({expirationGracePeriodDays: 0});
    }
    this.drawerRef = this.overlay.showDrawer(this.templateDrawerTemplate());
  }

  hideDrawer() {
    this.drawerRef?.close();
    this.selectedTemplate.set(undefined);
    this.templateForm.reset({expirationGracePeriodDays: 0});
  }

  async saveTemplate() {
    this.templateForm.markAllAsTouched();
    if (!this.templateForm.valid) {
      return;
    }
    this.templateFormLoading.set(true);
    const {name, payloadTemplate, expirationGracePeriodDays} = this.templateForm.getRawValue();
    const existing = this.selectedTemplate();
    const action = existing
      ? this.templatesService.update({...existing, name, payloadTemplate, expirationGracePeriodDays})
      : this.templatesService.create({name, payloadTemplate, expirationGracePeriodDays});
    try {
      const saved = await firstValueFrom(action);
      this.hideDrawer();
      this.toast.success(`${saved.name} saved successfully`);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.templateFormLoading.set(false);
    }
  }

  deleteTemplate(template: LicenseTemplate) {
    this.overlay
      .confirm(`Really delete template "${template.name}"?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.templatesService.delete(template)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return EMPTY;
        })
      )
      .subscribe();
  }
}
