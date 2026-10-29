import {DatePipe} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {Router, RouterLink} from '@angular/router';
import {DeploymentType} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDownload, faGear, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, map, of, startWith, Subject, switchMap, take} from 'rxjs';
import {downloadBlob} from '../../../util/blob';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {AlertComponent} from '../../components/alert/alert.component';
import {DeploymentTypeBadgeComponent} from '../../components/deployment-type-badge.component';
import {PageComponent} from '../../components/page.component';
import {SearchBarComponent} from '../../components/search-bar.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {ClickableRowDirective} from '../../directives/clickable-row.directive';
import {AuthService} from '../../services/auth.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService, supportBundleZipFileName} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundle} from '../../types/support-bundle';
import {supportBundleDeleteConfirm} from '../support-bundle-display';
import {SupportBundleStatusBadgeComponent} from '../support-bundle-status-badge.component';

@Component({
  selector: 'app-support-bundle-list',
  templateUrl: './support-bundle-list.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    DatePipe,
    ReactiveFormsModule,
    RouterLink,
    FaIconComponent,
    AutotrimDirective,
    ClickableRowDirective,
    PageComponent,
    SearchBarComponent,
    SupportBundleStatusBadgeComponent,
    AlertComponent,
    DeploymentTypeBadgeComponent,
  ],
})
export class SupportBundleListComponent {
  protected readonly auth = inject(AuthService);
  private readonly svc = inject(SupportBundlesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly router = inject(Router);

  protected readonly faDownload = faDownload;
  protected readonly faGear = faGear;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly routePrefix = this.auth.isCustomer() ? '/support' : '/support-bundles';

  protected readonly configExists = toSignal(
    this.auth.isVendor() ? this.svc.getConfiguration().pipe(map((envVars) => envVars.length > 0)) : of(false),
    {initialValue: false}
  );

  private readonly refresh$ = new Subject<void>();
  private readonly bundles$ = this.refresh$.pipe(
    startWith(0),
    switchMap(() => this.svc.list()),
    takeUntilDestroyed()
  );

  protected readonly filterForm = new FormGroup({
    search: new FormControl(''),
  });

  protected readonly filteredBundles = toSignal(
    filteredByFormControl(this.bundles$, this.filterForm.controls.search, (bundle: SupportBundle, search: string) => {
      const q = search.toLowerCase();
      return (
        (bundle.title || '').toLowerCase().includes(q) ||
        (bundle.customerOrganizationName || '').toLowerCase().includes(q) ||
        bundle.status.toLowerCase().includes(q) ||
        bundle.deploymentType.includes(q)
      );
    }).pipe(takeUntilDestroyed()),
    {initialValue: [] as SupportBundle[]}
  );

  private readonly bundlesResult = toSignal(this.bundles$);
  protected readonly bundles = computed(() => this.bundlesResult() ?? []);
  protected readonly loading = computed(() => this.bundlesResult() === undefined);

  // Create dialog (customer only)
  protected readonly createForm = new FormGroup({
    deploymentType: new FormControl<DeploymentType>('docker', {nonNullable: true}),
    title: new FormControl('', {nonNullable: true, validators: [Validators.required]}),
    description: new FormControl('', {nonNullable: true}),
  });
  protected createFormLoading = false;
  private dialogRef: DialogRef | null = null;

  openDialog(templateRef: TemplateRef<unknown>) {
    this.closeDialog();
    this.createForm.reset();
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
      const bundle = await firstValueFrom(
        this.svc.create({
          deploymentType: this.createForm.controls.deploymentType.value,
          title: this.createForm.value.title!,
          description: this.createForm.value.description || undefined,
        })
      );
      this.closeDialog();
      this.toast.success('Support bundle created');
      await this.router.navigate([this.routePrefix, bundle.id], {queryParams: {initialized: true}});
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.createFormLoading = false;
    }
  }

  protected readonly downloadingBundleId = signal<string | null>(null);

  protected async downloadResources(bundle: SupportBundle): Promise<void> {
    if (this.downloadingBundleId() !== null) {
      return;
    }
    this.downloadingBundleId.set(bundle.id);
    try {
      const blob = await firstValueFrom(this.svc.downloadResources(bundle.id));
      downloadBlob(blob, supportBundleZipFileName(bundle));
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      if (this.downloadingBundleId() === bundle.id) {
        this.downloadingBundleId.set(null);
      }
    }
  }

  protected readonly deletingBundleId = signal<string | null>(null);

  protected async deleteBundle(bundle: SupportBundle): Promise<void> {
    if (this.deletingBundleId() !== null) {
      return;
    }
    const confirmed = await firstValueFrom(this.overlay.confirm(supportBundleDeleteConfirm));
    if (!confirmed) {
      return;
    }
    this.deletingBundleId.set(bundle.id);
    try {
      await firstValueFrom(this.svc.delete(bundle.id));
      this.toast.success('Support bundle deleted');
      this.refresh$.next();
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      if (this.deletingBundleId() === bundle.id) {
        this.deletingBundleId.set(null);
      }
    }
  }
}
