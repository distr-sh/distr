import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, computed, inject, input, signal} from '@angular/core';
import {toObservable} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faThumbtack, faThumbtackSlash} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, EMPTY, filter, interval, map, merge, Observable, scan, Subject, switchMap, tap} from 'rxjs';
import {distinctBy} from '../../util/arrays';
import {downloadBlob} from '../../util/blob';
import {ToastService} from '../services/toast.service';
import {SpinnerComponent} from './spinner/spinner.component';

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
  template: `
    @if (entries$ | async; as entries) {
      <div class="relative flex flex-col gap-4" [class.flex-col-reverse]="!newestFirst()">
        @if (sourceWithStatus().loading()) {
          <div class="absolute top-0 inset-e-0 rounded-xl bg-white dark:bg-gray-900 z-10 p-2">
            <output class="flex justify-center items-center gap-2 text-gray-700 dark:text-gray-400 text-xs">
              <app-spinner class="size-5" />
              <span>Loading&hellip;</span>
            </output>
          </div>
        }
        <table class="w-full text-sm text-left rtl:text-right text-gray-500 dark:text-gray-400">
          <thead
            class="dark:border-gray-600 text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-800 dark:text-gray-400 sr-only">
            <tr>
              <th scope="col"></th>
              <th scope="col">Date</th>
              <th scope="col">Status</th>
              <th scope="col">Details</th>
            </tr>
          </thead>
          <tbody>
            @for (entry of entries; track entry.id ?? entry.date) {
              @let pinned = pinnedEntryId() === entry.id;
              <tr
                [class.bg-yellow-100]="pinned"
                [class.dark:bg-yellow-950]="pinned"
                [class.sticky]="pinned"
                [class.top-16]="pinned"
                [class.bottom-0]="pinned"
                [class.hover:bg-gray-50]="!pinned"
                [class.dark:hover:bg-gray-600]="!pinned"
                [class.hover:bg-yellow-200]="pinned"
                [class.dark:hover:bg-yellow-900]="pinned"
                class="not-last:border-b border-gray-200 dark:border-gray-600  group">
                <td class="w-0">
                  <button
                    type="button"
                    (click)="pin(entry)"
                    [class.invisible]="!pinned"
                    class="group-hover:visible px-1 text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 text-xs">
                    <fa-icon [icon]="pinned ? faThumbtackSlash : faThumbtack" />
                  </button>
                </td>
                <th class="px-1 font-medium whitespace-nowrap">{{ entry.date | date: 'medium' }}</th>
                <td
                  class="uppercase px-1"
                  [class.text-red-500]="entry.status.toLowerCase().includes('err')"
                  [class.dark:text-red-400]="entry.status.toLowerCase().includes('err')"
                  [class.text-yellow-600]="entry.status.toLowerCase().includes('warn')"
                  [class.dark:text-yellow-500]="entry.status.toLowerCase().includes('warn')">
                  {{ entry.status }}
                </td>
                <td class="px-1 max-w-0 w-full" data-ph-mask-text="true">
                  <div class="overflow-x-auto whitespace-pre-wrap font-mono text-gray-900 dark:text-white">
                    {{ entry.detail }}
                  </div>
                </td>
              </tr>
            }
          </tbody>
        </table>

        @if (hasOlder) {
          <div class="flex items-center justify-center gap-2">
            <button
              type="button"
              class="py-2 px-3 flex items-center text-sm font-medium text-center text-gray-900 focus:outline-none bg-white rounded-lg border border-gray-200 hover:bg-gray-100 hover:text-primary-700 focus:z-10 focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-600 dark:hover:text-white dark:hover:bg-gray-700"
              (click)="showOlder()">
              Load older
            </button>
          </div>
        }
      </div>
    } @else {
      <output class="flex justify-center items-center gap-2 text-gray-700 dark:text-gray-400">
        <app-spinner class="size-8" />
        <span>Loading&hellip;</span>
      </output>
    }
  `,
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
          source.load(),
          this.showOlder$.pipe(
            map(() => nextBefore),
            filter((before) => before !== null),
            switchMap((before) => source.loadBefore(before))
          )
        ).pipe(tap((entries) => (this.hasOlder = entries.length >= source.batchSize))),
        live
          ? interval(10_000).pipe(
              map(() => nextAfter),
              filter((after) => after !== null),
              switchMap((after) => source.loadAfter(after))
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
        toastRef?.toastRef.close();
        this.toastService.success('Download completed successfully');
      },
      error: (err) => {
        console.error('Export failed:', err);
        this.isExporting = false;
        toastRef?.toastRef.close();
        this.toastService.error('Export failed');
      },
    });
  }
}

function compareByDate(reverse: boolean): (a: TimeseriesEntry, b: TimeseriesEntry) => number {
  const mod = reverse ? -1 : 1;
  return (a, b) => mod * (new Date(a.date).getTime() - new Date(b.date).getTime());
}
