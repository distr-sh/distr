import {DatePipe, NgClass} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {Advisory, AdvisorySeverity, AdvisoryStatus, PatchAdvisoryRequest} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatest, firstValueFrom, of, shareReplay, startWith, Subject, switchMap, take} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {filteredByFormControl} from '../../util/filter';
import {BadgeSelectComponent} from '../components/badge-select/badge-select.component';
import {MultiSelectComponent, MultiSelectOption} from '../components/multi-select/multi-select.component';
import {PageComponent} from '../components/page.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {AdvisoriesService} from '../services/advisories.service';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {
  affectedBadgeClass,
  affectedLabel,
  confirmAdvisoryVisibilityChange,
  defaultAdvisoryStatusFilter,
  severityBadgeClass,
  severitySelectOptions,
  statusBadgeClass,
  statusLabel,
  statusSelectOptions,
} from './advisory-display';
import {AdvisoryFormComponent, AdvisoryFormDraft} from './advisory-form.component';

@Component({
  selector: 'app-advisory-list',
  templateUrl: './advisory-list.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    DatePipe,
    NgClass,
    RouterLink,
    FaIconComponent,
    AdvisoryFormComponent,
    BadgeSelectComponent,
    MultiSelectComponent,
    PageComponent,
    SearchBarComponent,
  ],
})
export class AdvisoryListComponent {
  protected readonly auth = inject(AuthService);
  private readonly advisoriesService = inject(AdvisoriesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  protected readonly faPlus = faPlus;
  protected readonly severityBadgeClass = severityBadgeClass;
  protected readonly statusBadgeClass = statusBadgeClass;
  protected readonly statusLabel = statusLabel;
  protected readonly affectedLabel = affectedLabel;
  protected readonly affectedBadgeClass = affectedBadgeClass;
  protected readonly statusSelectOptions = statusSelectOptions;
  protected readonly severitySelectOptions = severitySelectOptions;

  protected readonly showsAffectedState = this.auth.isCustomer() || this.auth.isPartner();

  protected readonly routePrefix = this.auth.isCustomer() ? '/security' : '/advisories';
  protected readonly canEdit = this.auth.isVendor() && this.auth.hasAnyRole('read_write', 'admin');
  protected readonly canFilterByStatus = !this.showsAffectedState;

  protected readonly filterForm = new FormGroup({
    search: new FormControl(''),
  });

  private readonly defaultStatuses: AdvisoryStatus[] = this.showsAffectedState ? [] : defaultAdvisoryStatusFilter;

  protected readonly selectedStatuses = signal<AdvisoryStatus[]>([...this.defaultStatuses]);
  protected readonly selectedSeverities = signal<AdvisorySeverity[]>([]);
  protected readonly selectedTags = signal<string[]>([]);

  // The tags endpoint spans undisclosed advisories and is therefore vendor only.
  private readonly vendorTags = toSignal(
    this.auth.isVendor()
      ? this.advisoriesService.listTags().pipe(catchError(() => of<string[]>([])))
      : of<string[]>([]),
    {initialValue: [] as string[]}
  );
  protected readonly tagOptions = computed<MultiSelectOption[]>(() => {
    const tags = this.auth.isVendor()
      ? this.vendorTags()
      : [...new Set(this.advisories().flatMap((advisory) => advisory.tags))].sort();
    return tags.map((tag) => ({value: tag, label: tag}));
  });

  private readonly searchValue = toSignal(this.filterForm.controls.search.valueChanges, {initialValue: ''});

  // Compared against the default rather than against an empty selection, so that an
  // organization with no advisories at all still gets the introductory empty state instead of
  // being told nothing matched.
  private readonly statusFilterChanged = computed(() => {
    const selected = this.selectedStatuses();
    return (
      selected.length !== this.defaultStatuses.length ||
      !this.defaultStatuses.every((status) => selected.includes(status))
    );
  });

  protected readonly filtersActive = computed(() =>
    Boolean(
      this.searchValue() || this.statusFilterChanged() || this.selectedSeverities().length || this.selectedTags().length
    )
  );

  private readonly refresh$ = new Subject<void>();

  private readonly serverFilter = computed(() => ({
    status: this.selectedStatuses(),
    severity: this.selectedSeverities(),
    tag: this.selectedTags(),
  }));

  // shareReplay keeps the two subscribers below from each issuing their own request.
  private readonly advisories$ = combineLatest([
    this.refresh$.pipe(startWith(0)),
    toObservable(this.serverFilter),
  ]).pipe(
    switchMap(([, filter]) => this.advisoriesService.list(filter)),
    shareReplay({bufferSize: 1, refCount: true}),
    takeUntilDestroyed()
  );

  // The free-text search stays client side: it spans several fields and the result set is
  // small enough that a round trip per keystroke is not worth it.
  protected readonly filteredAdvisories = toSignal(
    filteredByFormControl(this.advisories$, this.filterForm.controls.search, (advisory: Advisory, search: string) => {
      const query = search.toLowerCase();
      return (
        advisory.title.toLowerCase().includes(query) ||
        (advisory.cveId ?? '').toLowerCase().includes(query) ||
        (!this.showsAffectedState && advisory.status.includes(query)) ||
        advisory.severity.includes(query) ||
        advisory.tags.some((tag) => tag.toLowerCase().includes(query))
      );
    }).pipe(takeUntilDestroyed()),
    {initialValue: [] as Advisory[]}
  );

  private readonly advisoriesResult = toSignal(this.advisories$);
  protected readonly advisories = computed(() => this.advisoriesResult() ?? []);
  protected readonly loading = computed(() => this.advisoriesResult() === undefined);

  private readonly patchingFor = signal<string | undefined>(undefined);

  protected isPatching(advisoryId: string): boolean {
    return this.patchingFor() === advisoryId;
  }

  protected async changeStatus(advisory: Advisory, status: AdvisoryStatus): Promise<void> {
    if (await confirmAdvisoryVisibilityChange(this.overlay, advisory.status, status)) {
      await this.patch(advisory, {status}, 'Status updated');
    }
  }

  protected changeSeverity(advisory: Advisory, severity: AdvisorySeverity): Promise<void> {
    return this.patch(advisory, {severity}, 'Severity updated');
  }

  private async patch(advisory: Advisory, request: PatchAdvisoryRequest, success: string): Promise<void> {
    if (this.patchingFor()) {
      return;
    }
    this.patchingFor.set(advisory.id);
    try {
      await firstValueFrom(this.advisoriesService.patch(advisory.id, request));
      this.toast.success(success);
      this.refresh$.next();
    } catch (e) {
      const message = getFormDisplayedError(e);
      if (message) {
        this.toast.error(message);
      }
    } finally {
      this.patchingFor.set(undefined);
    }
  }

  private dialogRef: DialogRef | null = null;
  protected readonly dialogOpen = signal(false);
  protected readonly createDraft = signal<AdvisoryFormDraft | undefined>(undefined);

  protected openCreateDialog(templateRef: TemplateRef<unknown>): void {
    this.closeDialog();
    this.dialogOpen.set(true);
    this.dialogRef = this.overlay.showModal(templateRef);
    this.dialogRef
      .result()
      .pipe(take(1))
      .subscribe(() => {
        this.dialogRef = null;
        this.dialogOpen.set(false);
      });
  }

  protected closeDialog(): void {
    this.dialogRef?.dismiss();
    this.dialogRef = null;
    this.dialogOpen.set(false);
  }

  protected cancelCreateDialog(): void {
    this.createDraft.set(undefined);
    this.closeDialog();
  }

  protected onCreated(): void {
    this.createDraft.set(undefined);
    this.closeDialog();
    this.refresh$.next();
  }
}
