import {ChangeDetectionStrategy, Component, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {EditorComponent} from '../../components/editor.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleConfigurationScript} from '../../types/support-bundle';

@Component({
  selector: 'app-support-bundle-scripts',
  templateUrl: './support-bundle-scripts.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FaIconComponent, AutotrimDirective, EditorComponent],
  host: {class: 'block space-y-4'},
})
export class SupportBundleScriptsComponent {
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faPen = faPen;
  protected readonly faXmark = faXmark;

  protected readonly auth = inject(AuthService);
  private readonly fb = inject(FormBuilder).nonNullable;
  private readonly svc = inject(SupportBundlesService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);

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

  constructor() {
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
      name: this.scriptForm.controls.name.value,
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

  protected async toggleScriptEnabled(script: SupportBundleConfigurationScript, toggle: HTMLInputElement) {
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
      toggle.checked = script.enabled;
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
}
