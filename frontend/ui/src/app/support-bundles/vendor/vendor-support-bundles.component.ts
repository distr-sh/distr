import {HttpErrorResponse} from '@angular/common/http';
import {Component, inject, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormArray, FormBuilder, FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faFloppyDisk, faPlus, faTrash} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleConfigurationEnvVar} from '../../types/support-bundle';

type EnvVarFormGroup = FormGroup<{
  name: FormControl<string>;
  redacted: FormControl<boolean>;
}>;

@Component({
  selector: 'app-vendor-support-bundles',
  templateUrl: './vendor-support-bundles.component.html',
  imports: [ReactiveFormsModule, FaIconComponent],
})
export class VendorSupportBundlesComponent {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;

  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly svc = inject(SupportBundlesService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);

  protected readonly loading = signal(true);
  protected readonly saving = signal(false);
  protected readonly configExists = signal(false);

  protected readonly envVarsArray = new FormArray<EnvVarFormGroup>([]);

  constructor() {
    this.svc
      .getConfiguration()
      .pipe(takeUntilDestroyed())
      .subscribe({
        next: (config) => {
          this.configExists.set(true);
          for (const envVar of config.envVars) {
            this.addEnvVar(envVar);
          }
          this.loading.set(false);
        },
        error: (e) => {
          if (e instanceof HttpErrorResponse && e.status === 404) {
            this.configExists.set(false);
          } else {
            const msg = getFormDisplayedError(e);
            if (msg) {
              this.toast.error(msg);
            }
          }
          this.loading.set(false);
        },
      });
  }

  protected addEnvVar(envVar?: SupportBundleConfigurationEnvVar) {
    this.envVarsArray.push(
      this.fb.group({
        name: this.fb.control(envVar?.name ?? ''),
        redacted: this.fb.control(envVar?.redacted ?? false),
      })
    );
  }

  protected removeEnvVar(index: number) {
    this.envVarsArray.removeAt(index);
  }

  protected async save() {
    this.saving.set(true);
    const envVars: SupportBundleConfigurationEnvVar[] = this.envVarsArray.controls.map((group) => ({
      name: group.controls.name.value,
      redacted: group.controls.redacted.value,
    }));

    try {
      await firstValueFrom(this.svc.updateConfiguration({envVars}));
      this.configExists.set(true);
      this.toast.success('Support bundle configuration saved');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.saving.set(false);
    }
  }

  protected async deleteConfiguration() {
    const confirmed = await firstValueFrom(
      this.overlay.confirm('Are you sure you want to delete the support bundle configuration?')
    );
    if (!confirmed) {
      return;
    }

    try {
      await firstValueFrom(this.svc.deleteConfiguration());
      this.configExists.set(false);
      this.envVarsArray.clear();
      this.toast.success('Support bundle configuration deleted');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
