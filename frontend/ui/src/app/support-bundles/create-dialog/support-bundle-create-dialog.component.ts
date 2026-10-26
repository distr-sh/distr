import {Component, inject, output, signal} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {AlertComponent} from '../../components/alert/alert.component';
import {ClipComponent} from '../../components/clip.component';
import {ClosableDialog} from '../../components/confirm-dialog/closable-dialog';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';

@Component({
  selector: 'app-support-bundle-create-dialog',
  templateUrl: './support-bundle-create-dialog.component.html',
  imports: [ReactiveFormsModule, FaIconComponent, ClipComponent, AutotrimDirective, AlertComponent],
})
export class SupportBundleCreateDialogComponent extends ClosableDialog {
  private readonly svc = inject(SupportBundlesService);
  private readonly toast = inject(ToastService);

  public readonly created = output<void>();
  public readonly closed = output<void>();

  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;

  protected readonly form = new FormGroup({
    title: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
    description: new FormControl('', {nonNullable: true}),
  });
  protected readonly loading = signal(false);
  protected readonly collectCommand = signal<string | null>(null);

  protected async submit() {
    if (this.form.invalid) {
      return;
    }
    this.loading.set(true);
    try {
      const response = await firstValueFrom(
        this.svc.create({
          title: this.form.value.title!,
          description: this.form.value.description || undefined,
        })
      );
      this.collectCommand.set(response.collectCommand);
      this.toast.success('Support bundle created');
      this.created.emit();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.loading.set(false);
    }
  }
}
