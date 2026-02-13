import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, input} from '@angular/core';
import {toObservable} from '@angular/core/rxjs-interop';
import {filter, interval, map, merge, Observable, scan, Subject, switchMap, tap} from 'rxjs';
import {distinctBy} from '../../util/arrays';
import {downloadBlob} from '../../util/blob';
import {ToastService} from '../services/toast.service';

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

export interface TimeseriesExporter {
  getFileName(): string;
  export(): Observable<Blob>;
}

@Component({
  selector: 'app-timeseries-table',
  template: `
    @if (entries$ | async; as entries) {
      <div class="relative overflow-x-auto">
        <table class="w-full text-sm text-left rtl:text-right text-gray-500 dark:text-gray-400">
          <thead
            class="dark:border-gray-600 text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-800 dark:text-gray-400 sr-only">
            <tr>
              <th scope="col">Date</th>
              <th scope="col">Status</th>
              <th scope="col">Details</th>
            </tr>
          </thead>
          <tbody>
            @for (entry of entries; track entry.id ?? entry.date) {
              <tr
                class="not-last:border-b border-gray-200 dark:border-gray-600 hover:bg-gray-50 dark:hover:bg-gray-600">
                <th class="px-4 md:px-5 font-medium whitespace-nowrap">
                  {{ entry.date | date: 'medium' }}
                </th>
                <td class="uppercase">
                  {{ entry.status }}
                </td>
                <td
                  class="px-4 md:px-5 whitespace-pre-wrap font-mono text-gray-900 dark:text-white"
                  data-ph-mask-text="true">
                  {{ entry.detail }}
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>

      @if (hasMore || exporter()) {
        <div class="flex items-center justify-center gap-2 mt-2">
          @if (hasMore) {
            <button
              type="button"
              class="py-2 px-3 flex items-center text-sm font-medium text-center text-gray-900 focus:outline-none bg-white rounded-lg border border-gray-200 hover:bg-gray-100 hover:text-primary-700 focus:z-10 focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-600 dark:hover:text-white dark:hover:bg-gray-700"
              (click)="showMore()">
              Load more
            </button>
          }
          @if (exporter()) {
            <button
              type="button"
              class="py-2 px-3 flex items-center text-sm font-medium text-center text-gray-900 focus:outline-none bg-white rounded-lg border border-gray-200 hover:bg-gray-100 hover:text-primary-700 focus:z-10 focus:ring-4 focus:ring-gray-200 dark:focus:ring-gray-700 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-600 dark:hover:text-white dark:hover:bg-gray-700"
              (click)="exportData()"
              [disabled]="isExporting">
              @if (isExporting) {
                Exporting...
              } @else {
                Export all
              }
            </button>
          }
        </div>
      }
    } @else {
      <output class="flex justify-center items-center gap-2 text-gray-700 dark:text-gray-400">
        <svg
          aria-hidden="true"
          class="w-8 h-8 text-gray-200 animate-spin dark:text-gray-600 fill-blue-600"
          viewBox="0 0 100 101"
          fill="none"
          xmlns="http://www.w3.org/2000/svg">
          <path
            d="M100 50.5908C100 78.2051 77.6142 100.591 50 100.591C22.3858 100.591 0 78.2051 0 50.5908C0 22.9766 22.3858 0.59082 50 0.59082C77.6142 0.59082 100 22.9766 100 50.5908ZM9.08144 50.5908C9.08144 73.1895 27.4013 91.5094 50 91.5094C72.5987 91.5094 90.9186 73.1895 90.9186 50.5908C90.9186 27.9921 72.5987 9.67226 50 9.67226C27.4013 9.67226 9.08144 27.9921 9.08144 50.5908Z"
            fill="currentColor" />
          <path
            d="M93.9676 39.0409C96.393 38.4038 97.8624 35.9116 97.0079 33.5539C95.2932 28.8227 92.871 24.3692 89.8167 20.348C85.8452 15.1192 80.8826 10.7238 75.2124 7.41289C69.5422 4.10194 63.2754 1.94025 56.7698 1.05124C51.7666 0.367541 46.6976 0.446843 41.7345 1.27873C39.2613 1.69328 37.813 4.19778 38.4501 6.62326C39.0873 9.04874 41.5694 10.4717 44.0505 10.1071C47.8511 9.54855 51.7191 9.52689 55.5402 10.0491C60.8642 10.7766 65.9928 12.5457 70.6331 15.2552C75.2735 17.9648 79.3347 21.5619 82.5849 25.841C84.9175 28.9121 86.7997 32.2913 88.1811 35.8758C89.083 38.2158 91.5421 39.6781 93.9676 39.0409Z"
            fill="currentFill" />
        </svg>
        <span>Loading&hellip;</span>
      </output>
    }
  `,
  imports: [DatePipe, AsyncPipe],
})
export class TimeseriesTableComponent {
  public readonly source = input.required<TimeseriesSource>();
  public readonly exporter = input<TimeseriesExporter>();

  private readonly toastService = inject(ToastService);

  protected hasMore = true;
  protected isExporting = false;

  protected readonly entries$: Observable<TimeseriesEntry[]> = toObservable(this.source).pipe(
    switchMap((source) => {
      let nextBefore: Date | null = null;
      let nextAfter: Date | null = null;
      return merge(
        merge(
          source.load(),
          this.showMore$.pipe(
            map(() => nextBefore),
            filter((before) => before !== null),
            switchMap((before) => source.loadBefore(before))
          )
        ).pipe(tap((entries) => (this.hasMore = entries.length >= source.batchSize))),
        interval(10_000).pipe(
          map(() => nextAfter),
          filter((after) => after !== null),
          switchMap((after) => source.loadAfter(after))
        )
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
          (acc, entries) =>
            distinctBy((it: TimeseriesEntry) => it.id ?? it.date)(acc.concat(entries)).sort(compareByDate),
          [] as TimeseriesEntry[]
        )
      );
    })
  );

  private readonly showMore$ = new Subject<void>();

  protected showMore() {
    this.showMore$.next();
  }

  protected exportData() {
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
        toastRef.toastRef.close();
        this.toastService.success('Download completed successfully');
      },
      error: (err) => {
        console.error('Export failed:', err);
        this.isExporting = false;
        toastRef.toastRef.close();
        this.toastService.error('Export failed');
      },
    });
  }
}

function compareByDate(a: TimeseriesEntry, b: TimeseriesEntry): number {
  return new Date(b.date).getTime() - new Date(a.date).getTime();
}
