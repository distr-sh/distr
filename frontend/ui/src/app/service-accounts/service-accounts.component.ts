import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe, TitleCasePipe} from '@angular/common';
import {Component, inject, input, TemplateRef} from '@angular/core';
import {toObservable} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {AccountRole, CreateServiceAccountRequest, ServiceAccount} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faKey, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, firstValueFrom, map, startWith, Subject, switchMap} from 'rxjs';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ServiceAccountsService} from '../services/service-accounts.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-service-accounts',
  imports: [
    AsyncPipe,
    AutotrimDirective,
    DatePipe,
    FaIconComponent,
    OverlayModule,
    ReactiveFormsModule,
    RouterLink,
    TitleCasePipe,
  ],
  templateUrl: './service-accounts.component.html',
})
export class ServiceAccountsComponent {
  public readonly customerOrganizationId = input<string | undefined>();

  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly faKey = faKey;

  private readonly service = inject(ServiceAccountsService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  private readonly refresh$ = new Subject<void>();
  private readonly customerOrganizationId$ = toObservable(this.customerOrganizationId);
  protected readonly serviceAccounts$ = combineLatest([
    this.refresh$.pipe(
      startWith(0),
      switchMap(() => this.service.list())
    ),
    this.customerOrganizationId$,
  ]).pipe(map(([sas, customerOrgId]) => this.filterByCustomer(sas, customerOrgId)));

  protected drawer: DialogRef<void> | null = null;
  protected createLoading = false;

  protected readonly createForm = new FormGroup({
    name: new FormControl<string>('', {nonNullable: true, validators: [Validators.required]}),
    accountRole: new FormControl<AccountRole>('read_write', {nonNullable: true, validators: [Validators.required]}),
  });

  public openCreateDrawer(template: TemplateRef<unknown>) {
    this.hideDrawer();
    this.createForm.reset({name: '', accountRole: 'read_write'});
    this.drawer = this.overlay.showDrawer(template);
  }

  public hideDrawer() {
    this.drawer?.dismiss();
  }

  public async create() {
    if (this.createForm.invalid) {
      return;
    }
    this.createLoading = true;
    try {
      const request: CreateServiceAccountRequest = {
        name: this.createForm.value.name!,
        accountRole: this.createForm.value.accountRole!,
      };
      if (this.customerOrganizationId()) {
        request.customerOrganizationId = this.customerOrganizationId();
      }
      await firstValueFrom(this.service.create(request));
      this.toast.success('Service account created');
      this.hideDrawer();
      this.refresh$.next();
    } finally {
      this.createLoading = false;
    }
  }

  public async delete(sa: ServiceAccount) {
    if (await firstValueFrom(this.overlay.confirm(`Really delete service account '${sa.name}'?`))) {
      try {
        await firstValueFrom(this.service.delete(sa.id));
        this.refresh$.next();
      } catch {
        // toast handled globally
      }
    }
  }

  private filterByCustomer(sas: ServiceAccount[], customerOrgId: string | undefined): ServiceAccount[] {
    if (customerOrgId) {
      return sas.filter((sa) => sa.customerOrganizationId === customerOrgId);
    }
    return sas.filter((sa) => !sa.customerOrganizationId);
  }
}
