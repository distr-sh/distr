import {HttpErrorResponse} from '@angular/common/http';
import {Component, inject, OnInit, signal} from '@angular/core';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk, faPaperPlane, faTrash} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {HOSTNAME_MAX_LENGTH, HOSTNAME_REGEX} from '../../util/validation';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {CustomEmailService} from '../services/custom-email.service';
import {OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {CustomEmailConfiguration, CustomEmailSettings} from '../types/custom-email';

@Component({
  selector: 'app-custom-email',
  templateUrl: './custom-email.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective],
})
export class CustomEmailComponent implements OnInit {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faPaperPlane = faPaperPlane;
  protected readonly faTrash = faTrash;

  private readonly customEmailService = inject(CustomEmailService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly configuration = signal<CustomEmailConfiguration | undefined>(undefined);
  protected readonly saveLoading = signal(false);
  protected readonly testLoading = signal(false);

  protected readonly form = this.fb.group({
    fromAddress: this.fb.control('', [Validators.required, Validators.email]),
    smtpHost: this.fb.control('', [
      Validators.required,
      Validators.pattern(HOSTNAME_REGEX),
      Validators.maxLength(HOSTNAME_MAX_LENGTH),
    ]),
    smtpPort: this.fb.control(587, [Validators.required, Validators.min(1), Validators.max(65535)]),
    smtpUsername: this.fb.control(''),
    smtpPassword: this.fb.control(''),
    smtpImplicitTls: this.fb.control(false),
    enabled: this.fb.control(true),
  });

  async ngOnInit() {
    try {
      this.applyConfiguration(await firstValueFrom(this.customEmailService.get()));
    } catch (e) {
      // An organization without an email configuration is the normal case, so 404 is not an error.
      if (e instanceof HttpErrorResponse && e.status === 404) {
        return;
      }
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async save() {
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      return;
    }
    this.saveLoading.set(true);
    try {
      this.applyConfiguration(
        await firstValueFrom(
          this.customEmailService.upsert({...this.settings(), enabled: this.form.getRawValue().enabled})
        )
      );
      this.toast.success('Email configuration saved successfully');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.saveLoading.set(false);
    }
  }

  protected async sendTest() {
    this.form.markAllAsTouched();
    if (this.form.invalid) {
      return;
    }
    this.testLoading.set(true);
    try {
      await firstValueFrom(this.customEmailService.test(this.settings()));
      this.toast.success('Test email sent, check your inbox');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.testLoading.set(false);
    }
  }

  protected async remove() {
    const confirmed = await firstValueFrom(
      this.overlay.confirm('Really remove the email configuration? Emails will be sent by Distr again.')
    );
    if (!confirmed) {
      return;
    }
    try {
      await firstValueFrom(this.customEmailService.delete());
      this.configuration.set(undefined);
      this.form.reset();
      this.toast.success('Email configuration removed');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  /**
   * The password is only sent when it was entered, so that saving another setting does not clear
   * the stored one. It is never sent back to the browser.
   */
  private settings(): CustomEmailSettings {
    const value = this.form.getRawValue();
    return {
      fromAddress: value.fromAddress,
      smtpHost: value.smtpHost.toLowerCase(),
      smtpPort: value.smtpPort,
      smtpUsername: value.smtpUsername,
      smtpPassword: value.smtpPassword || undefined,
      smtpImplicitTls: value.smtpImplicitTls,
    };
  }

  private applyConfiguration(configuration: CustomEmailConfiguration) {
    this.configuration.set(configuration);
    this.form.reset({
      fromAddress: configuration.fromAddress,
      smtpHost: configuration.smtpHost,
      smtpPort: configuration.smtpPort,
      smtpUsername: configuration.smtpUsername,
      smtpPassword: '',
      smtpImplicitTls: configuration.smtpImplicitTls,
      enabled: configuration.enabled,
    });
  }
}
