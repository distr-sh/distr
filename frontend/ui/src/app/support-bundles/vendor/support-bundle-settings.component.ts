import {Component, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {
  AbstractControl,
  FormArray,
  FormBuilder,
  FormControl,
  FormGroup,
  ReactiveFormsModule,
  ValidationErrors,
  ValidatorFn,
} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft, faFileImport, faFloppyDisk, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleConfigurationEnvVar} from '../../types/support-bundle';

type EnvVarFormGroup = FormGroup<{
  name: FormControl<string>;
  redacted: FormControl<boolean>;
}>;

function uniqueNamesValidator(): ValidatorFn {
  return (control: AbstractControl): ValidationErrors | null => {
    const array = control as FormArray<EnvVarFormGroup>;
    const seen = new Map<string, number>();
    const dupes = new Set<number>();
    for (let i = 0; i < array.length; i++) {
      const name = array.at(i).controls.name.value.trim().toUpperCase();
      if (!name) continue;
      const prev = seen.get(name);
      if (prev !== undefined) {
        dupes.add(prev);
        dupes.add(i);
      } else {
        seen.set(name, i);
      }
    }
    return dupes.size > 0 ? {duplicateNames: Array.from(dupes)} : null;
  };
}

@Component({
  selector: 'app-support-bundle-settings',
  templateUrl: './support-bundle-settings.component.html',
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, RouterLink],
})
export class SupportBundleSettingsComponent {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faFileImport = faFileImport;
  protected readonly faXmark = faXmark;
  protected readonly faArrowLeft = faArrowLeft;

  protected readonly auth = inject(AuthService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly svc = inject(SupportBundlesService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);

  protected readonly loading = signal(true);
  protected readonly saving = signal(false);

  protected readonly envVarsArray = new FormArray<EnvVarFormGroup>([], {validators: uniqueNamesValidator()});

  protected get duplicateIndices(): Set<number> {
    const errors = this.envVarsArray.errors;
    if (errors?.['duplicateNames']) {
      return new Set(errors['duplicateNames'] as number[]);
    }
    return new Set();
  }

  constructor() {
    this.svc
      .getConfiguration()
      .pipe(takeUntilDestroyed())
      .subscribe({
        next: (envVars) => {
          for (const envVar of envVars) {
            this.addEnvVar(envVar);
          }
          this.envVarsArray.markAsPristine();
          this.loading.set(false);
        },
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
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
      name: group.controls.name.value.trim(),
      redacted: group.controls.redacted.value,
    }));

    try {
      await firstValueFrom(this.svc.updateConfiguration({envVars}));
      this.envVarsArray.markAsPristine();
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
      this.envVarsArray.clear();
      this.envVarsArray.markAsPristine();
      this.toast.success('Support bundle configuration deleted');
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected readonly importText = new FormControl('', {nonNullable: true});
  private importModalRef?: DialogRef;

  protected openImportModal(templateRef: TemplateRef<unknown>) {
    this.importText.reset();
    this.importModalRef = this.overlay.showModal(templateRef);
  }

  protected closeImportModal() {
    this.importModalRef?.dismiss();
    this.importModalRef = undefined;
  }

  protected importEnvVars() {
    const existingNames = new Set(this.envVarsArray.controls.map((g) => g.controls.name.value.trim().toUpperCase()));
    const lines = this.importText.value.split('\n');
    let added = 0;
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) {
        continue;
      }
      const match = trimmed.match(/^([^=:]+)/);
      if (!match) {
        continue;
      }
      const name = match[1].trim();
      if (!name || existingNames.has(name.toUpperCase())) {
        continue;
      }
      existingNames.add(name.toUpperCase());
      this.addEnvVar({name, redacted: false});
      added++;
    }
    if (added > 0) {
      this.toast.success(`Imported ${added} variable${added > 1 ? 's' : ''}`);
    }
    this.closeImportModal();
  }
}
