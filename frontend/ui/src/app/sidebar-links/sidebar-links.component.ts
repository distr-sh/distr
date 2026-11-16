import {DatePipe} from '@angular/common';
import {Component, computed, inject, input, TemplateRef, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {SidebarLink} from '@distr-sh/distr-sdk';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {SpinnerComponent} from '../components/spinner/spinner.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ClickableRowDirective} from '../directives/clickable-row.directive';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {SidebarLinksService} from '../services/sidebar-links.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-sidebar-links',
  templateUrl: './sidebar-links.component.html',
  imports: [
    FontAwesomeModule,
    ReactiveFormsModule,
    DatePipe,
    AutotrimDirective,
    ClickableRowDirective,
    SpinnerComponent,
  ],
})
export class SidebarLinksComponent {
  protected readonly faXmark = faXmark;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;

  protected readonly canEdit = inject(AuthService).hasAnyRole('read_write', 'admin');
  private readonly linksService = inject(SidebarLinksService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  public readonly customerOrganizationId = input.required<string>();

  private readonly refresh$ = new Subject<void>();

  protected readonly links = toSignal(
    combineLatest([toObservable(this.customerOrganizationId), this.refresh$.pipe(startWith(undefined))]).pipe(
      switchMap(([customerOrganizationId]) => this.linksService.list(customerOrganizationId))
    )
  );
  protected readonly loaded = computed(() => this.links() !== undefined);

  private readonly createUpdateDialog = viewChild.required<TemplateRef<unknown>>('createUpdateDialog');
  private dialogRef?: DialogRef;

  protected readonly createUpdateForm = this.fb.group({
    id: this.fb.control(''),
    name: this.fb.control('', [Validators.required]),
    link: this.fb.control('', [Validators.required]),
  });

  protected closeDialog() {
    this.createUpdateForm.reset();
    this.dialogRef?.close();
  }

  protected showDialog(existing?: SidebarLink) {
    this.closeDialog();

    if (existing) {
      this.createUpdateForm.setValue({
        id: existing.id,
        name: existing.name,
        link: existing.link,
      });
    }

    this.dialogRef = this.overlay.showModal(this.createUpdateDialog());
  }

  protected async submitForm() {
    this.createUpdateForm.markAllAsTouched();
    if (!this.createUpdateForm.valid) {
      return;
    }

    const customerOrganizationId = this.customerOrganizationId();
    const {id, name, link} = this.createUpdateForm.getRawValue();

    try {
      if (id) {
        await firstValueFrom(this.linksService.update(customerOrganizationId, id, name, link));
        this.toast.success('Link has been updated.');
      } else {
        await firstValueFrom(this.linksService.create(customerOrganizationId, name, link));
        this.toast.success('Link has been created.');
      }
      this.refresh$.next();
      this.closeDialog();
    } catch (error) {
      const msg = getFormDisplayedError(error);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  protected async deleteLink(link: SidebarLink) {
    if (await firstValueFrom(this.overlay.confirm(`Do you really want to delete the link "${link.name}"?`))) {
      try {
        await firstValueFrom(this.linksService.delete(this.customerOrganizationId(), link.id));
        this.refresh$.next();
      } catch (error) {
        const msg = getFormDisplayedError(error);
        if (msg) {
          this.toast.error(msg);
        }
      }
    }
  }
}
