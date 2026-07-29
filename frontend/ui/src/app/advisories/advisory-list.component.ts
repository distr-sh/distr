import {DatePipe, NgClass} from '@angular/common';
import {ChangeDetectionStrategy, Component, computed, inject, signal, TemplateRef} from '@angular/core';
import {takeUntilDestroyed, toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faMagnifyingGlass, faPlus} from '@fortawesome/free-solid-svg-icons';
import {catchError, combineLatest, firstValueFrom, of, shareReplay, startWith, Subject, switchMap, take} from 'rxjs';
import {getFormDisplayedError} from '../../util/errors';
import {filteredByFormControl} from '../../util/filter';
import {MultiSelectComponent, MultiSelectOption} from '../components/multi-select/multi-select.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AdvisoriesService} from '../services/advisories.service';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {Advisory, AdvisorySeverity, AdvisoryStatus} from '../types/advisory';
import {
  advisorySeverities,
  advisoryStatuses,
  defaultAdvisoryStatusFilter,
  quickStatusTransitionsFor,
  severityBadgeClass,
  severityLabel,
  statusActionShortLabel,
  statusBadgeClass,
  statusChangeConfirmation,
  statusLabel,
} from './advisory-display';
import {AdvisoryFormComponent, AdvisoryFormDraft} from './advisory-form.component';

@Component({
  selector: 'app-advisory-list',
  templateUrl: './advisory-list.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [
    DatePipe,
    NgClass,
    ReactiveFormsModule,
    RouterLink,
    FaIconComponent,
    AutotrimDirective,
    AdvisoryFormComponent,
    MultiSelectComponent,
  ],
})
export class AdvisoryListComponent {
  protected readonly auth = inject(AuthService);
  private readonly advisoriesService = inject(AdvisoriesService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPlus = faPlus;
  protected readonly severityBadgeClass = severityBadgeClass;
  protected readonly statusBadgeClass = statusBadgeClass;
  protected readonly statusActionShortLabel = statusActionShortLabel;
  protected readonly quickStatusTransitionsFor = quickStatusTransitionsFor;

  protected readonly routePrefix = this.auth.isCustomer() ? '/security' : '/advisories';
  protected readonly canEdit = this.auth.isVendor() && this.auth.hasAnyRole('read_write', 'admin');
  // Customers only ever see published and resolved items, so a status filter adds nothing.
  protected readonly canFilterByStatus = !this.auth.isCustomer();

  protected readonly filterForm = new FormGroup({
    search: new FormControl(''),
  });

  // Customers have no status dropdown, so sending them a status filter would be noise; the
  // backend already limits them to published and resolved.
  private readonly defaultStatuses: AdvisoryStatus[] = this.canFilterByStatus ? defaultAdvisoryStatusFilter : [];

  protected readonly selectedStatuses = signal<AdvisoryStatus[]>([...this.defaultStatuses]);
  protected readonly selectedSeverities = signal<AdvisorySeverity[]>([]);
  protected readonly selectedTags = signal<string[]>([]);

  protected readonly statusOptions: MultiSelectOption<AdvisoryStatus>[] = advisoryStatuses.map((status) => ({
    value: status,
    label: statusLabel(status),
  }));
  protected readonly severityOptions: MultiSelectOption<AdvisorySeverity>[] = advisorySeverities.map((severity) => ({
    value: severity,
    label: severityLabel(severity),
  }));

  // The tags endpoint is vendor and partner only, so customers must not request it.
  protected readonly tags = toSignal(
    this.auth.isCustomer() ? of<string[]>([]) : this.advisoriesService.listTags().pipe(catchError(() => of([]))),
    {initialValue: [] as string[]}
  );
  protected readonly tagOptions = computed<MultiSelectOption[]>(() =>
    this.tags().map((tag) => ({value: tag, label: tag}))
  );

  private readonly searchValue = toSignal(this.filterForm.controls.search.valueChanges, {initialValue: ''});

  // Compared against the default rather than against an empty selection, so that an
  // organization with no advisories at all still gets the introductory empty state
  // instead of being told nothing matched.
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

  // Recomputes only when a selection actually changes, so the list is not refetched on every
  // unrelated signal write.
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
        advisory.status.includes(query) ||
        advisory.severity.includes(query) ||
        advisory.tags.some((tag) => tag.toLowerCase().includes(query))
      );
    }).pipe(takeUntilDestroyed()),
    {initialValue: [] as Advisory[]}
  );

  private readonly advisoriesResult = toSignal(this.advisories$);
  protected readonly advisories = computed(() => this.advisoriesResult() ?? []);
  protected readonly loading = computed(() => this.advisoriesResult() === undefined);

  /** Id of the row whose status is currently being changed, so only its buttons disable. */
  private readonly changingStatusFor = signal<string | undefined>(undefined);

  protected isChangingStatus(advisoryId: string): boolean {
    return this.changingStatusFor() === advisoryId;
  }

  protected async changeStatus(advisory: Advisory, status: AdvisoryStatus): Promise<void> {
    if (this.changingStatusFor()) {
      return;
    }

    const confirmation = statusChangeConfirmation(status);
    if (confirmation && !(await firstValueFrom(this.overlay.confirm(confirmation)))) {
      return;
    }

    this.changingStatusFor.set(advisory.id);
    try {
      await firstValueFrom(this.advisoriesService.updateStatus(advisory.id, {status}));
      this.toast.success('Status updated');
      this.refresh$.next();
    } catch (e) {
      const message = getFormDisplayedError(e);
      if (message) {
        this.toast.error(message);
      }
    } finally {
      this.changingStatusFor.set(undefined);
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
