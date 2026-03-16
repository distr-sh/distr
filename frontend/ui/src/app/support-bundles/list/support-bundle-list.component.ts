import {DatePipe, NgClass} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGear, faMagnifyingGlass, faPlus, faXmark} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, map, of, startWith, Subject, switchMap, take} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {never} from '../../../util/exhaust';
import {filteredByFormControl} from '../../../util/filter';
import {ClipComponent} from '../../components/clip.component';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {AuthService} from '../../services/auth.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundle, SupportBundleStatus} from '../../types/support-bundle';

@Component({
  selector: 'app-support-bundle-list',
  templateUrl: './support-bundle-list.component.html',
  imports: [DatePipe, NgClass, ReactiveFormsModule, RouterLink, FaIconComponent, ClipComponent, AutotrimDirective],
})
export class SupportBundleListComponent {
  protected readonly auth = inject(AuthService);
  private readonly svc = inject(SupportBundlesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  protected readonly faGear = faGear;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;

  protected readonly routePrefix = this.auth.isVendor() ? '/support-bundles' : '/support';

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
        bundle.status.toLowerCase().includes(q)
      );
    }).pipe(takeUntilDestroyed()),
    {initialValue: [] as SupportBundle[]}
  );

  private readonly bundlesResult = toSignal(this.bundles$);
  protected readonly bundles = computed(() => this.bundlesResult() ?? []);
  protected readonly loading = computed(() => this.bundlesResult() === undefined);

  // Create dialog (customer only)
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
        this.svc.create({
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

  protected statusBadgeClass(status: SupportBundleStatus): string {
    switch (status) {
      case 'initialized':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
      case 'created':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300';
      case 'resolved':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
      case 'canceled':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
      default:
        return never(status);
    }
  }
}
