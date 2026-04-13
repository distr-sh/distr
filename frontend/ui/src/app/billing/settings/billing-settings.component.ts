import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft, faCheck, faCircleXmark, faCopy} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {ClipDirective} from '../../components/clip.component';
import {AuthService} from '../../services/auth.service';
import {OrganizationService} from '../../services/organization.service';
import {OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';

@Component({
  selector: 'app-billing-settings',
  templateUrl: './billing-settings.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, ClipDirective, RouterLink],
})
export class BillingSettingsComponent {
  protected readonly auth = inject(AuthService);
  private readonly organizationService = inject(OrganizationService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faArrowLeft = faArrowLeft;
  protected readonly faCheck = faCheck;
  protected readonly faCircleXmark = faCircleXmark;
  protected readonly faCopy = faCopy;

  protected readonly organization = toSignal(this.organizationService.get());
  protected readonly webhookSecretLoading = signal(false);

  protected readonly webhookForm = this.fb.group({
    webhookSecret: this.fb.control('', [Validators.required, Validators.pattern(/^whsec_[a-zA-Z0-9]+$/)]),
  });

  protected readonly webhookUrl = computed(() => `${location.origin}/api/v1/webhook/${this.organization()?.id}/stripe`);

  async saveWebhookSecret() {
    this.webhookForm.markAllAsTouched();
    if (!this.webhookForm.valid) {
      return;
    }
    this.webhookSecretLoading.set(true);
    const {webhookSecret} = this.webhookForm.getRawValue();
    try {
      await firstValueFrom(this.organizationService.updateWebhookSecret(webhookSecret));
      this.webhookForm.reset();
      this.toast.success('Webhook secret saved successfully');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.webhookSecretLoading.set(false);
    }
  }

  async deleteWebhookSecret() {
    const confirmed = await firstValueFrom(this.overlay.confirm('Really remove the webhook secret?'));
    if (!confirmed) {
      return;
    }
    this.webhookSecretLoading.set(true);
    try {
      await firstValueFrom(this.organizationService.deleteWebhookSecret());
      this.toast.success('Webhook secret removed');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.webhookSecretLoading.set(false);
    }
  }
}
