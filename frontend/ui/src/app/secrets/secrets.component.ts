import {DatePipe} from '@angular/common';
import {Component, computed, inject, input, output, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faMagnifyingGlass, faPen, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {ClipComponent} from '../components/clip.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {SecretsService} from '../services/secrets.service';
import {ToastService} from '../services/toast.service';
import {AffectedDeployment} from '../types/affected-deployment';
import {Secret} from '../types/secret';

@Component({
  selector: 'app-secrets',
  imports: [FaIconComponent, ReactiveFormsModule, DatePipe, AutotrimDirective, ClipComponent],
  templateUrl: './secrets.component.html',
})
export class SecretsComponent {
  public readonly secrets = input.required<Secret[]>();
  public readonly refresh = output();

  protected readonly auth = inject(AuthService);
  private readonly overlay = inject(OverlayService);
  private readonly secretsService = inject(SecretsService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faXmark = faXmark;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faPen = faPen;

  private readonly createUpdateDialog = viewChild.required<TemplateRef<unknown>>('createUpdateDialog');
  private readonly affectedDeploymentsDialog = viewChild.required<TemplateRef<unknown>>('affectedDeploymentsDialog');
  private dialogRef?: DialogRef;
  protected readonly affectedDeployments = signal<AffectedDeployment[]>([]);

  protected readonly filterForm = this.fb.group({
    search: '',
  });

  private readonly filterValue = toSignal(this.filterForm.controls.search.valueChanges);

  protected readonly filteredSecrets = computed(() => {
    const value = this.filterValue()?.toLowerCase();
    const secrets = this.secrets();
    return !value ? secrets : secrets.filter((secret) => secret.key.toLowerCase().includes(value));
  });

  protected readonly createUpdateForm = this.fb.group({
    id: this.fb.control(''),
    key: this.fb.control('', [Validators.required, Validators.minLength(1), Validators.pattern('^[a-zA-Z][\\w_]*$')]),
    value: this.fb.control('', [Validators.required]),
  });

  protected closeDialog() {
    this.createUpdateForm.reset();
    this.dialogRef?.close();
  }

  protected showDialog(existingSecret?: Secret) {
    this.closeDialog();

    if (existingSecret) {
      this.createUpdateForm.setValue({
        id: existingSecret.id,
        key: existingSecret.key,
        value: '',
      });
      this.createUpdateForm.controls.key.disable();
    } else {
      this.createUpdateForm.controls.key.enable();
    }

    this.dialogRef = this.overlay.showModal(this.createUpdateDialog());
  }

  protected createSecret() {
    this.createUpdateForm.markAllAsTouched();
    if (!this.createUpdateForm.valid) return;

    const {id, key, value} = this.createUpdateForm.value;

    if (!id) {
      this.secretsService.create(key!, value!, this.customerOrganizationId()).subscribe({
        next: () => {
          this.toast.success('Secret has been created.');
          this.refresh.emit();
          this.closeDialog();
        },
        error: (error) => {
          const msg = getFormDisplayedError(error);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
    } else {
      void this.updateSecret(id, value!);
    }
  }

  private async updateSecret(id: string, value: string) {
    try {
      const dryRun = await firstValueFrom(this.secretsService.update(id, value, true));
      if (dryRun.affectedDeployments.length > 0) {
        this.affectedDeployments.set(dryRun.affectedDeployments);
        const confirmed = await firstValueFrom(
          this.overlay.confirm({
            customTemplate: this.affectedDeploymentsDialog(),
            confirmLabel: 'Update and redeploy',
          })
        );
        if (!confirmed) {
          return;
        }
      }

      await firstValueFrom(this.secretsService.update(id, value));
      this.toast.success('Secret value has been updated.');
      this.refresh.emit();
      this.closeDialog();
    } catch (error) {
      const msg = getFormDisplayedError(error);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.affectedDeployments.set([]);
    }
  }

  protected getSecretReference(key: string): string {
    return `{{ .Secrets.${key} }}`;
  }

  protected async deleteSecret(secret: Secret) {
    if (
      await firstValueFrom(
        this.overlay.confirm({
          message: {
            message: 'Do you really want to delete this secret?',
            alert: {type: 'warning', message: 'This action may affect workloads referencing this secret.'},
          },
          requiredConfirmInputText: secret.key,
        })
      )
    ) {
      try {
        await firstValueFrom(this.secretsService.delete(secret.id));
        this.refresh.emit();
      } catch (error) {
        const msg = getFormDisplayedError(error);
        if (msg) {
          this.toast.error(msg);
        }
      }
    }
  }
}
