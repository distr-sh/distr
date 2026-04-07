import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, computed, inject, input, signal} from '@angular/core';
import {toObservable} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faThumbtack, faThumbtackSlash} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  combineLatest,
  EMPTY,
  filter,
  interval,
  map,
  merge,
  Observable,
  scan,
  Subject,
  switchMap,
  tap,
} from 'rxjs';
import {distinctBy} from '../../../util/arrays';
import {downloadBlob} from '../../../util/blob';
import {getFormDisplayedError} from '../../../util/errors';
import {ToastService} from '../../services/toast.service';
import {SpinnerComponent} from '../spinner/spinner.component';

export interface TimeseriesEntry {
  id?: string;
  date: string;
  status: string;
  detail: string;
}

export interface TimeseriesSource {
  readonly batchSize: number;
  load(): Observable<TimeseriesEntry[]>;
  loadBefore(before: Date): Observable<TimeseriesEntry[]>;
  loadAfter(after: Date): Observable<TimeseriesEntry[]>;
}

export class TimeseriesSourceWithStatus implements TimeseriesSource {
  public readonly batchSize: number;
  private readonly loadingRW = signal(false);
  public readonly loading = this.loadingRW.asReadonly();

  constructor(private readonly source: TimeseriesSource) {
    this.batchSize = source.batchSize;
  }

  load(): Observable<TimeseriesEntry[]> {
    this.loadingRW.set(true);
    return this.source.load().pipe(
      tap({
        finalize: () => this.loadingRW.set(false),
      })
    );
  }

  loadBefore(before: Date): Observable<TimeseriesEntry[]> {
    this.loadingRW.set(true);
    return this.source.loadBefore(before).pipe(
      tap({
        finalize: () => this.loadingRW.set(false),
      })
    );
  }

  loadAfter(after: Date): Observable<TimeseriesEntry[]> {
    this.loadingRW.set(true);
    return this.source.loadAfter(after).pipe(
      tap({
        finalize: () => this.loadingRW.set(false),
      })
    );
  }
}

export interface TimeseriesExporter {
  getFileName(): string;
  export(): Observable<Blob>;
}

@Component({
  selector: 'app-timeseries-table',
  templateUrl: './timeseries-table.component.html',
  imports: [DatePipe, AsyncPipe, FaIconComponent, SpinnerComponent],
})
export class TimeseriesTableComponent {
  public readonly source = input.required<TimeseriesSource>();
  public readonly exporter = input<TimeseriesExporter>();
  public readonly live = input<boolean>(true);
  public readonly newestFirst = input<boolean>(true);

  private readonly toastService = inject(ToastService);

  protected readonly faThumbtack = faThumbtack;
  protected readonly faThumbtackSlash = faThumbtackSlash;

  protected hasOlder = true;
  protected isExporting = false;
  protected readonly pinnedEntryId = signal<string | null>(null);

  protected readonly sourceWithStatus = computed(() => new TimeseriesSourceWithStatus(this.source()));

  private readonly accumulatedEntries$: Observable<TimeseriesEntry[]> = combineLatest([
    toObservable(this.sourceWithStatus),
    toObservable(this.live),
  ]).pipe(
    switchMap(([source, live]) => {
      let nextBefore: Date | null = null;
      let nextAfter: Date | null = null;
      return merge(
        merge(
          source.load().pipe(catchError((err) => this.handleError(err))),
          this.showOlder$.pipe(
            map(() => nextBefore),
            filter((before) => before !== null),
            switchMap((before) => source.loadBefore(before).pipe(catchError((err) => this.handleError(err))))
          )
        ).pipe(tap((entries) => (this.hasOlder = entries.length >= source.batchSize))),
        live
          ? interval(10_000).pipe(
              map(() => nextAfter),
              filter((after) => after !== null),
              switchMap((after) => source.loadAfter(after).pipe(catchError((err) => this.handleError(err))))
            )
          : EMPTY
      ).pipe(
        tap((entries) =>
          entries
            .map((entry) => new Date(entry.date))
            .forEach((ts) => {
              if (nextBefore === null || ts < nextBefore) {
                nextBefore = ts;
              }
              if (nextAfter === null || ts > nextAfter) {
                nextAfter = ts;
              }
            })
        ),
        scan(
          (acc, entries) => distinctBy((it: TimeseriesEntry) => it.id ?? it.date)(acc.concat(entries)),
          [] as TimeseriesEntry[]
        )
      );
    })
  );

  protected readonly entries$: Observable<TimeseriesEntry[]> = combineLatest([
    this.accumulatedEntries$,
    toObservable(this.newestFirst),
  ]).pipe(map(([entries, newestFirst]) => entries.sort(compareByDate(newestFirst))));

  private readonly showOlder$ = new Subject<void>();

  protected showOlder() {
    this.showOlder$.next();
  }

  private handleError(err: unknown) {
    const msg = getFormDisplayedError(err);
    if (msg) {
      this.toastService.error('Failed to load entries: ' + msg);
    } else {
      this.toastService.error('Failed to load entries');
    }
    return EMPTY;
  }

  protected pin(entry: TimeseriesEntry) {
    this.pinnedEntryId.update((current) => (current === entry.id ? null : entry.id) ?? null);
  }

  public exportData() {
    const exporter = this.exporter();
    if (!exporter) {
      return;
    }

    this.isExporting = true;

    const today = new Date().toISOString().split('T')[0];
    const filename = `${today}_${exporter.getFileName()}`;
    const toastRef = this.toastService.info('Download started...');

    exporter.export().subscribe({
      next: (blob) => {
        downloadBlob(blob, filename);
        this.isExporting = false;
        toastRef?.close();
        this.toastService.success('Download completed successfully');
      },
      error: (err) => {
        console.error('Export failed:', err);
        this.isExporting = false;
        toastRef?.close();
        this.toastService.error('Export failed');
      },
    });
  }
}

function compareByDate(reverse: boolean): (a: TimeseriesEntry, b: TimeseriesEntry) => number {
  const mod = reverse ? -1 : 1;
  return (a, b) => mod * (new Date(a.date).getTime() - new Date(b.date).getTime());
}
