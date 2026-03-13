import {DatePipe} from '@angular/common';
import {Component, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith, Subject, switchMap, take} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {ClipComponent} from '../../components/clip.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';

@Component({
  selector: 'app-customer-support-bundles',
  templateUrl: './customer-support-bundles.component.html',
  imports: [DatePipe, ReactiveFormsModule, RouterLink, FaIconComponent, ClipComponent, AutotrimDirective],
})
export class CustomerSupportBundlesComponent {
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  protected readonly auth = inject(AuthService);

  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;

  private readonly refresh$ = new Subject<void>();
  protected readonly bundles = toSignal(
    this.refresh$.pipe(
      startWith(0),
      switchMap(() => this.supportBundlesService.list()),
      takeUntilDestroyed()
    ),
    {initialValue: []}
  );

  protected readonly createForm = new FormGroup({
    title: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
    description: new FormControl('', {nonNullable: true}),
  });

  protected createFormLoading = false;
  protected collectCommand = signal<string | null>(null);

  private dialogRef: DialogRef | null = null;

  openDialog(templateRef: TemplateRef<unknown>) {
    this.closeDialog();
    this.createForm.reset();
    this.collectCommand.set(null);
    this.dialogRef = this.overlay.showModal(templateRef);
    this.dialogRef
      .result()
      .pipe(take(1))
      .subscribe(() => this.refresh$.next());
  }

  closeDialog() {
    this.dialogRef?.dismiss();
    this.dialogRef = null;
  }

  async createBundle() {
    if (this.createForm.invalid) {
      return;
    }
    this.createFormLoading = true;
    try {
      const response = await firstValueFrom(
        this.supportBundlesService.create({
          title: this.createForm.value.title!,
          description: this.createForm.value.description || undefined,
        })
      );
      this.collectCommand.set(response.collectCommand);
      this.toast.success('Support bundle created');
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.createFormLoading = false;
    }
  }
}
