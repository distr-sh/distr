import {ChangeDetectionStrategy, Component, inject, signal, TemplateRef} from '@angular/core';
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
  Validators,
} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faArrowLeft,
  faFileImport,
  faFloppyDisk,
  faPen,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {EditorComponent} from '../../components/editor.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleConfigurationEnvVar, SupportBundleConfigurationScript} from '../../types/support-bundle';

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
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, RouterLink, EditorComponent],
})
export class SupportBundleSettingsComponent {
  protected readonly faFloppyDisk = faFloppyDisk;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faPen = faPen;
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
    this.loadScripts();
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
    this.envVarsArray.markAsDirty();
  }

  protected async save() {
    if (await this.persistConfiguration()) {
      this.toast.success('Support bundle configuration saved');
    }
  }

  private async persistConfiguration(): Promise<boolean> {
    this.saving.set(true);
    const envVars: SupportBundleConfigurationEnvVar[] = this.envVarsArray.controls.map((group) => ({
      name: group.controls.name.value.trim(),
      redacted: group.controls.redacted.value,
    }));

    try {
      await firstValueFrom(this.svc.updateConfiguration({envVars}));
      this.envVarsArray.markAsPristine();
      return true;
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
      return false;
    } finally {
      this.saving.set(false);
    }
  }

  protected readonly scripts = signal<SupportBundleConfigurationScript[]>([]);
  protected readonly savingScript = signal(false);
  protected readonly scriptForm = this.fb.group({
    name: this.fb.control('', Validators.required),
    description: this.fb.control(''),
    content: this.fb.control('', Validators.required),
    enabled: this.fb.control(true),
  });
  private editedScript?: SupportBundleConfigurationScript;
  private scriptModalRef?: DialogRef;

  private loadScripts() {
    this.svc
      .getScripts()
      .pipe(takeUntilDestroyed())
      .subscribe({
        next: (scripts) => this.scripts.set(scripts),
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
  }

  protected openScriptModal(templateRef: TemplateRef<unknown>, script?: SupportBundleConfigurationScript) {
    this.editedScript = script;
    this.scriptForm.reset({
      name: script?.name ?? '',
      description: script?.description ?? '',
      content: script?.content ?? '#!/bin/sh\n\n',
      enabled: script?.enabled ?? true,
    });
    this.scriptModalRef = this.overlay.showModal(templateRef);
  }

  protected closeScriptModal() {
    this.scriptModalRef?.dismiss();
    this.scriptModalRef = undefined;
    this.editedScript = undefined;
  }

  protected async saveScript() {
    this.scriptForm.markAllAsTouched();
    if (!this.scriptForm.valid) {
      return;
    }
    const description = this.scriptForm.controls.description.value.trim();
    const request = {
      name: this.scriptForm.controls.name.value.trim(),
      description: description || undefined,
      content: this.scriptForm.controls.content.value,
      enabled: this.scriptForm.controls.enabled.value,
    };

    this.savingScript.set(true);
    try {
      const edited = this.editedScript;
      if (edited) {
        const updated = await firstValueFrom(this.svc.updateScript(edited.id, request));
        this.scripts.update((scripts) => scripts.map((s) => (s.id === updated.id ? updated : s)));
      } else {
        const created = await firstValueFrom(this.svc.createScript(request));
        this.scripts.update((scripts) => [...scripts, created]);
      }
      this.sortScripts();
      this.closeScriptModal();
      this.toast.success(`Script ${edited ? 'updated' : 'created'}`);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.savingScript.set(false);
    }
  }

  protected async toggleScriptEnabled(script: SupportBundleConfigurationScript) {
    try {
      const updated = await firstValueFrom(
        this.svc.updateScript(script.id, {
          name: script.name,
          description: script.description,
          content: script.content,
          enabled: !script.enabled,
        })
      );
      this.scripts.update((scripts) => scripts.map((s) => (s.id === updated.id ? updated : s)));
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async deleteScript(script: SupportBundleConfigurationScript) {
    const confirmed = await firstValueFrom(
      this.overlay.confirm(`Really delete the script "${script.name}"? It will no longer run for new support bundles.`)
    );
    if (!confirmed) {
      return;
    }
    try {
      await firstValueFrom(this.svc.deleteScript(script.id));
      this.scripts.update((scripts) => scripts.filter((s) => s.id !== script.id));
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  // The collect script runs the scripts ordered by name, so the list has to show them in that order.
  private sortScripts() {
    this.scripts.update((scripts) => [...scripts].sort((a, b) => a.name.localeCompare(b.name)));
  }

  protected readonly importText = new FormControl('', {nonNullable: true});
  private importModalRef?: DialogRef;

  protected openImportModal(templateRef: TemplateRef<unknown>) {
    if (this.envVarsArray.dirty) {
      return;
    }
    this.importText.reset();
    this.importModalRef = this.overlay.showModal(templateRef);
  }

  protected closeImportModal() {
    this.importModalRef?.dismiss();
    this.importModalRef = undefined;
  }

  protected async importEnvVars() {
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
    this.closeImportModal();
    if (added > 0) {
      this.envVarsArray.markAsDirty();
    }
    if (added > 0 && (await this.persistConfiguration())) {
      this.toast.success(`Imported and saved ${added} variable${added > 1 ? 's' : ''}`);
    }
  }
}
