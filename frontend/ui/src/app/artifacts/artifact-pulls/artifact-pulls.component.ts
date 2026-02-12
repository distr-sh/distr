import {DatePipe} from '@angular/common';
import {Component, DestroyRef, inject, signal} from '@angular/core';
import {takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {ActivatedRoute, Params, Router} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDownload, faFilterCircleXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {debounceTime, first, map, of, scan, shareReplay, startWith, Subject, switchMap, tap} from 'rxjs';
import {downloadBlob} from '../../../util/blob';
import {formatRemoteAddress} from '../../../util/format';
import {ArtifactPullFilters, ArtifactPullsService} from '../../services/artifact-pulls.service';
import {ToastService} from '../../services/toast.service';

@Component({
  templateUrl: './artifact-pulls.component.html',
  imports: [DatePipe, ReactiveFormsModule, FaIconComponent],
})
export class ArtifactPullsComponent {
  private readonly pullsService = inject(ArtifactPullsService);
  private readonly toast = inject(ToastService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);

  protected readonly faDownload = faDownload;
  protected readonly faFilterCircleXmark = faFilterCircleXmark;
  protected readonly today = dayjs().format('YYYY-MM-DD');
  protected readonly hasMore = signal(true);
  protected readonly isExporting = signal(false);
  private currentOldestPull?: Date;
  private readonly fetchCount = 50;
  private readonly showMore$ = new Subject<void>();
  private initializing = true;

  protected readonly filterForm = new FormGroup({
    customerOrganizationId: new FormControl(''),
    userAccountId: new FormControl(''),
    remoteAddress: new FormControl(''),
    artifactId: new FormControl(''),
    artifactVersionId: new FormControl(''),
    from: new FormControl(''),
    to: new FormControl(''),
  });

  protected readonly filterOptions = toSignal(this.pullsService.getFilterOptions());

  protected readonly versionOptions = toSignal(
    this.filterForm.controls.artifactId.valueChanges.pipe(
      startWith(this.filterForm.controls.artifactId.value),
      switchMap((artifactId) => {
        if (!this.initializing) {
          this.filterForm.controls.artifactVersionId.setValue('', {emitEvent: false});
        }
        if (artifactId) {
          return this.pullsService.getVersionOptions(artifactId);
        }
        return of([]);
      })
    ),
    {initialValue: []}
  );

  private readonly filters$ = this.filterForm.valueChanges.pipe(
    startWith(this.filterForm.value),
    debounceTime(300),
    tap(() => (this.initializing = false)),
    tap((values) => this.syncQueryParams(values)),
    map((values) => this.buildFilters(values)),
    shareReplay(1)
  );

  protected readonly pulls = toSignal(
    this.filters$.pipe(
      switchMap((filters) => {
        this.currentOldestPull = undefined;
        this.hasMore.set(true);
        return this.showMore$.pipe(
          startWith(undefined),
          switchMap(() =>
            this.pullsService.get({
              ...filters,
              before: this.currentOldestPull,
              count: this.fetchCount,
            })
          ),
          tap((it) => {
            if (it.length > 0) {
              this.currentOldestPull = new Date(it[it.length - 1].createdAt);
            }
            if (it.length < this.fetchCount) {
              this.hasMore.set(false);
            }
          }),
          scan((all, next) => [...all, ...next])
        );
      })
    ),
    {initialValue: []}
  );

  constructor() {
    this.initFromQueryParams();
  }

  protected showMore() {
    this.showMore$.next();
  }

  protected resetFilters() {
    this.filterForm.reset({
      customerOrganizationId: '',
      userAccountId: '',
      remoteAddress: '',
      artifactId: '',
      artifactVersionId: '',
      from: '',
      to: '',
    });
  }

  protected exportCsv() {
    this.isExporting.set(true);
    const toastRef = this.toast.info('Download started...');
    const filters = this.buildFilters(this.filterForm.value);
    this.pullsService.export(filters).subscribe({
      next: (blob) => {
        downloadBlob(blob, `${dayjs().format('YYYY-MM-DD')}_artifact_pulls.csv`);
        this.isExporting.set(false);
        toastRef.toastRef.close();
        this.toast.success('Download completed successfully');
      },
      error: () => {
        this.isExporting.set(false);
        toastRef.toastRef.close();
        this.toast.error('Export failed');
      },
    });
  }

  protected formatVersionName(name: string): string {
    const shaPrefix = 'sha256:';
    if (name.startsWith(shaPrefix)) {
      return name.substring(0, 17);
    }
    return name;
  }

  protected readonly formatRemoteAddress = formatRemoteAddress;

  private initFromQueryParams() {
    const params = this.route.snapshot.queryParams;
    const hasParams = Object.keys(this.filterForm.controls).some((key) => params[key]);
    if (!hasParams) {
      return;
    }

    // Wait for filter options to load, then apply query params.
    // This ensures <select> elements have their <option> children
    // before we set a value, preventing Angular from resetting them.
    this.pullsService
      .getFilterOptions()
      .pipe(first(), takeUntilDestroyed(this.destroyRef))
      .subscribe(() => {
        // Use setTimeout to let Angular complete the render cycle
        // so that <option> elements from @for are in the DOM
        setTimeout(() => {
          const patch: Record<string, string> = {};
          for (const key of Object.keys(this.filterForm.controls)) {
            if (params[key]) {
              patch[key] = params[key];
            }
          }
          this.filterForm.patchValue(patch);
        });
      });
  }

  private syncQueryParams(values: typeof this.filterForm.value) {
    const queryParams: Params = {};
    for (const [key, value] of Object.entries(values)) {
      queryParams[key] = value || null;
    }
    this.router.navigate([], {queryParams, replaceUrl: true});
  }

  private buildFilters(values: typeof this.filterForm.value): ArtifactPullFilters {
    const filters: ArtifactPullFilters = {};
    if (values.customerOrganizationId) {
      filters.customerOrganizationId = values.customerOrganizationId;
    }
    if (values.userAccountId) {
      filters.userAccountId = values.userAccountId;
    }
    if (values.remoteAddress) {
      filters.remoteAddress = values.remoteAddress;
    }
    if (values.artifactId) {
      filters.artifactId = values.artifactId;
    }
    if (values.artifactVersionId) {
      filters.artifactVersionId = values.artifactVersionId;
    }
    const from = values.from;
    const to = values.to;
    if (from && to && dayjs(from).isAfter(dayjs(to))) {
      return filters;
    }
    if (from) {
      filters.after = dayjs(from).startOf('day').toDate();
    }
    if (to) {
      filters.before = dayjs(to).endOf('day').toDate();
    }
    return filters;
  }
}
