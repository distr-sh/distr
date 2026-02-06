import {Component, inject, OnInit, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk, faLightbulb} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, lastValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {slugMaxLength, slugPattern} from '../../util/slug';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {FeatureFlagService} from '../services/feature-flag.service';
import {OrganizationService} from '../services/organization.service';
import {OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {Organization} from '../types/organization';

@Component({
  selector: 'app-organization-settings',
  templateUrl: './organization-settings.component.html',
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective],
})
export class OrganizationSettingsComponent implements OnInit {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faLightbulb = faLightbulb;

  private readonly organizationService = inject(OrganizationService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly ff = inject(FeatureFlagService);
  private readonly overlayService = inject(OverlayService);
  protected readonly auth = inject(AuthService);

  protected readonly isPrePostScriptEnabled = toSignal(this.ff.isPrePostScriptEnabled$);

  private organization?: Organization;

  protected readonly form = this.fb.group({
    name: this.fb.control('', [Validators.required]),
    slug: this.fb.control('', [Validators.pattern(slugPattern), Validators.maxLength(slugMaxLength)]),
    appDomain: this.fb.control<string | undefined>({value: undefined, disabled: true}),
    registryDomain: this.fb.control<string | undefined>({value: undefined, disabled: true}),
    emailFromAddress: this.fb.control<string | undefined>({value: undefined, disabled: true}),
    preConnectScript: this.fb.control<string | undefined>(undefined),
    postConnectScript: this.fb.control<string | undefined>(undefined),
    connectScriptIsSudo: this.fb.control<boolean>(false),
    artifactVersionMutable: this.fb.control<boolean>(false),
  });
  formLoading = signal(false);

  async ngOnInit() {
    try {
      this.organization = await firstValueFrom(this.organizationService.get());
      if (this.organization.slug) {
        this.form.controls.slug.addValidators([Validators.required]);
      }
      this.form.patchValue({
        ...this.organization,
        artifactVersionMutable: this.organization.features?.includes('artifact_version_mutable') ?? false,
      });
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
      this.formLoading.set(true);
      try {
        this.organization = await lastValueFrom(
          this.organizationService.update({
            ...this.organization!,
            name: this.form.value.name?.trim()!,
            slug: this.form.value.slug?.trim(),
            preConnectScript: this.form.value.preConnectScript?.trim(),
            postConnectScript: this.form.value.postConnectScript?.trim(),
            connectScriptIsSudo: this.form.value.connectScriptIsSudo ?? false,
            artifactVersionMutable: this.form.value.artifactVersionMutable ?? false,
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

  async deleteOrganization() {
    try {
      if (
        await firstValueFrom(
          this.overlayService.confirm({
            message: {
              message:
                'Are you sure you want to delete this organization? ' +
                'Afterwards, all user sessions (including the current one) will be invalidated ' +
                'and users will be redirected to the login page.',
              alert: {
                type: 'danger',
                message: 'This is a destructive action and cannot be undone!',
              },
            },
            requiredConfirmInputText: `DELETE ${this.organization!.name.toUpperCase()}`,
          })
        )
      ) {
        const email = this.auth.getClaims()?.email;
        await firstValueFrom(this.organizationService.delete());
        await firstValueFrom(this.auth.logout());
        location.assign(`/login?email=${encodeURIComponent(email ?? '')}`);
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
