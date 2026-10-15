import {ChangeDetectionStrategy, Component, inject, OnInit, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {DeploymentType} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {slugMaxLength, slugPattern} from '../../util/slug';
import {DeleteOrganizationComponent} from '../components/delete-organization/delete-organization.component';
import {PillTabBarComponent} from '../components/pill-tab-bar.component';
import {TabItem} from '../components/tab-bar.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {OrganizationService} from '../services/organization.service';
import {OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {Organization} from '../types/organization';

interface ConnectScriptSettings {
  prePostScriptsEnabled: boolean;
  preConnectScript?: string;
  postConnectScript?: string;
  connectScriptIsSudo: boolean;
  kubernetesPreConnectScript?: string;
  kubernetesPostConnectScript?: string;
}

/** Mirrors usesConnectScript in internal/controllerconnect. */
function deploymentTypesUsingConnectScript(settings: ConnectScriptSettings): DeploymentType[] {
  if (!settings.prePostScriptsEnabled) {
    return [];
  }
  const isSet = (script?: string) => !!script?.trim();
  const result: DeploymentType[] = [];
  if (isSet(settings.preConnectScript) || isSet(settings.postConnectScript) || settings.connectScriptIsSudo) {
    result.push('docker');
  }
  if (isSet(settings.kubernetesPreConnectScript) || isSet(settings.kubernetesPostConnectScript)) {
    result.push('kubernetes');
  }
  return result;
}

@Component({
  selector: 'app-general-settings',
  templateUrl: './general-settings.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [FaIconComponent, ReactiveFormsModule, AutotrimDirective, DeleteOrganizationComponent, PillTabBarComponent],
})
export class GeneralSettingsComponent implements OnInit {
  protected readonly faFloppyDisk = faFloppyDisk;

  private readonly organizationService = inject(OrganizationService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly overlayService = inject(OverlayService);
  protected readonly auth = inject(AuthService);

  private readonly preflightConfirmTemplate = viewChild.required<TemplateRef<unknown>>('preflightConfirmTemplate');

  private organization?: Organization;

  protected readonly form = this.fb.group({
    name: this.fb.control('', [Validators.required]),
    slug: this.fb.control('', [Validators.pattern(slugPattern), Validators.maxLength(slugMaxLength)]),
    preConnectScript: this.fb.control<string | undefined>(undefined),
    postConnectScript: this.fb.control<string | undefined>(undefined),
    connectScriptIsSudo: this.fb.control<boolean>(false),
    kubernetesPreConnectScript: this.fb.control<string | undefined>(undefined),
    kubernetesPostConnectScript: this.fb.control<string | undefined>(undefined),
    artifactVersionMutable: this.fb.control<boolean>(false),
    prePostScriptsEnabled: this.fb.control<boolean>(false),
  });
  formLoading = signal(false);

  protected readonly connectScriptTabs: TabItem<DeploymentType>[] = [
    {id: 'docker', label: 'Docker'},
    {id: 'kubernetes', label: 'Kubernetes'},
  ];
  protected readonly connectScriptTab = signal<DeploymentType>('docker');
  protected readonly newlyWrappedDeploymentTypes = signal<DeploymentType[]>([]);

  protected readonly isPrePostScriptEnabled = toSignal(
    this.form.controls.prePostScriptsEnabled.valueChanges.pipe(
      startWith(this.form.controls.prePostScriptsEnabled.value)
    ),
    {initialValue: false}
  );

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
      const formValue = this.form.getRawValue();
      const wrappedBefore = deploymentTypesUsingConnectScript({
        ...this.organization!,
        prePostScriptsEnabled: this.organization?.features?.includes('pre_post_scripts') ?? false,
      });
      const newlyWrapped = deploymentTypesUsingConnectScript(formValue).filter((t) => !wrappedBefore.includes(t));

      if (newlyWrapped.length > 0) {
        this.newlyWrappedDeploymentTypes.set(newlyWrapped);
        const confirmed = await firstValueFrom(
          this.overlayService.confirm({
            customTemplate: this.preflightConfirmTemplate(),
            message: {
              message: '',
              alert: {
                type: 'warning',
                message:
                  'Existing controllers are not affected. New controller connect commands will use the new format.',
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
            ...formValue,
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
}
