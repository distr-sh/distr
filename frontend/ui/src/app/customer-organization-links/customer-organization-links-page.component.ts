import {OverlayModule} from '@angular/cdk/overlay';
import {DatePipe} from '@angular/common';
import {Component, computed, ElementRef, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {SidebarLink} from '@distr-sh/distr-sdk';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {
  faBoxesStacked,
  faChevronDown,
  faMagnifyingGlass,
  faPen,
  faPlus,
  faTrash,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, of, startWith, Subject, switchMap} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AuthService} from '../services/auth.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {SidebarLinksService} from '../services/sidebar-links.service';
import {ToastService} from '../services/toast.service';

@Component({
  templateUrl: './customer-organization-links-page.component.html',
  imports: [RouterLink, FontAwesomeModule, OverlayModule, ReactiveFormsModule, DatePipe, AutotrimDirective],
})
export class SidebarLinksPageComponent {
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faChevronDown = faChevronDown;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faXmark = faXmark;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faPen = faPen;

  protected readonly auth = inject(AuthService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly linksService = inject(SidebarLinksService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly fb = inject(FormBuilder).nonNullable;

  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );
  protected readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly customerOrganization = computed(() => {
    const id = this.customerOrganizationId();
    return this.customerOrganizations()?.find((org) => org.id === id);
  });

  protected readonly refresh$ = new Subject<void>();

  protected readonly links = toSignal(
    combineLatest([this.refresh$.pipe(startWith(undefined)), toObservable(this.customerOrganizationId)]).pipe(
      switchMap(([, customerOrganizationId]) => {
        if (!customerOrganizationId) {
          return of([]);
        }
        return this.linksService.list(customerOrganizationId);
      })
    )
  );

  protected readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly breadcrumbDropdown = signal(false);
  breadcrumbDropdownWidth = 0;

  private readonly createUpdateDialog = viewChild.required<TemplateRef<unknown>>('createUpdateDialog');
  private dialogRef?: DialogRef;

  protected readonly createUpdateForm = this.fb.group({
    id: this.fb.control(''),
    name: this.fb.control('', [Validators.required]),
    link: this.fb.control('', [Validators.required]),
  });

  protected toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((v) => !v);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth = this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width;
    }
  }

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

  protected submitForm() {
    this.createUpdateForm.markAllAsTouched();
    if (!this.createUpdateForm.valid) return;

    const customerOrgId = this.customerOrganizationId();
    if (!customerOrgId) return;

    const {id, name, link} = this.createUpdateForm.value;

    if (!id) {
      this.linksService.create(customerOrgId, name!, link!).subscribe({
        next: () => {
          this.toast.success('Link has been created.');
          this.refresh$.next();
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
      this.linksService.update(customerOrgId, id, name!, link!).subscribe({
        next: () => {
          this.toast.success('Link has been updated.');
          this.refresh$.next();
          this.closeDialog();
        },
        error: (error) => {
          const msg = getFormDisplayedError(error);
          if (msg) {
            this.toast.error(msg);
          }
        },
      });
    }
  }

  protected async deleteLink(link: SidebarLink) {
    const customerOrgId = this.customerOrganizationId();
    if (!customerOrgId) return;

    if (
      await firstValueFrom(
        this.overlay.confirm({
          message: {
            message: `Do you really want to delete the link "${link.name}"?`,
          },
        })
      )
    ) {
      try {
        await firstValueFrom(this.linksService.delete(customerOrgId, link.id));
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
