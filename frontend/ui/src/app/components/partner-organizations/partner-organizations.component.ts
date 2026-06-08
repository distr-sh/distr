import {DecimalPipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {PartnerOrganizationWithUsage} from '@distr-sh/distr-sdk';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faBuilding, faEdit, faMagnifyingGlass, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {filter, firstValueFrom, map, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {PartnerOrganizationsService} from '../../services/partner-organizations.service';
import {ToastService} from '../../services/toast.service';

@Component({
  templateUrl: './partner-organizations.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [ReactiveFormsModule, FontAwesomeModule, DecimalPipe, RouterLink],
})
export class PartnerOrganizationsComponent {
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPlus = faPlus;
  protected readonly faBuilding = faBuilding;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faEdit = faEdit;

  private readonly partnerOrganizationsService = inject(PartnerOrganizationsService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly filterForm = this.fb.group({
    search: this.fb.control(''),
  });

  private readonly refresh$ = new Subject<void>();
  protected readonly partnerOrganizations = toSignal(
    this.filterForm.controls.search.valueChanges.pipe(
      startWith(''),
      switchMap((search) =>
        this.refresh$.pipe(
          startWith(undefined),
          switchMap(() => this.partnerOrganizationsService.getPartnerOrganizations()),
          map((orgs) =>
            search.length > 0 ? orgs.filter((o) => o.name.toLowerCase().includes(search.toLowerCase())) : orgs
          )
        )
      )
    )
  );

  private readonly createPartnerDialog = viewChild.required<TemplateRef<unknown>>('createPartnerDialog');
  private modalRef?: DialogRef;
  protected readonly createForm = this.fb.group({
    id: this.fb.control(''),
    name: this.fb.control('', [Validators.required]),
  });
  protected createFormLoading = signal(false);

  protected showCreateDialog() {
    this.closeCreateDialog();
    this.modalRef = this.overlay.showModal(this.createPartnerDialog());
  }

  protected showUpdateDialog(partner: PartnerOrganizationWithUsage) {
    this.closeCreateDialog();
    this.createForm.patchValue(partner);
    this.modalRef = this.overlay.showModal(this.createPartnerDialog());
  }

  protected closeCreateDialog(reset = true) {
    this.modalRef?.close();
    if (reset) {
      this.createForm.reset();
    }
  }

  protected async submitCreateForm() {
    this.createForm.markAllAsTouched();
    if (this.createForm.invalid) {
      return;
    }
    this.createFormLoading.set(true);
    try {
      const name = this.createForm.value.name!;
      if (this.createForm.value.id) {
        await firstValueFrom(
          this.partnerOrganizationsService.updatePartnerOrganization(this.createForm.value.id, {name})
        );
      } else {
        await firstValueFrom(this.partnerOrganizationsService.createPartnerOrganization({name}));
      }
      this.closeCreateDialog();
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.createFormLoading.set(false);
    }
  }

  protected delete(partner: PartnerOrganizationWithUsage) {
    this.overlay
      .confirm({
        message: {
          message: 'Are you sure you want to delete this partner?',
          alert:
            partner.userCount > 0 || partner.customerOrganizationCount > 0
              ? {
                  type: 'warning',
                  message: [
                    partner.userCount > 0
                      ? `Deleting this partner will remove its ${partner.userCount} user(s) from your organization.`
                      : null,
                    partner.customerOrganizationCount > 0
                      ? `Its ${partner.customerOrganizationCount} customer(s) will not be deleted but will be unassigned.`
                      : null,
                  ]
                    .filter(Boolean)
                    .join(' '),
                }
              : undefined,
        },
        requiredConfirmInputText: partner.name,
      })
      .pipe(
        filter((it) => it === true),
        switchMap(() => this.partnerOrganizationsService.deletePartnerOrganization(partner.id))
      )
      .subscribe({
        next: () => this.refresh$.next(),
        error: (e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
  }
}
