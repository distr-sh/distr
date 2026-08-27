import {AsyncPipe, DatePipe, NgClass, NgTemplateOutlet} from '@angular/common';
import {
  afterNextRender,
  Component,
  computed,
  ElementRef,
  inject,
  Injector,
  input,
  signal,
  viewChild,
} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowDown, faThumbtack, faThumbtackSlash} from '@fortawesome/free-solid-svg-icons';
import {
  catchError,
  combineLatest,
  EMPTY,
  exhaustMap,
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
import {IntersectionObserverDirective} from '../../directives/intersection-observer.directive';
import {ToastService} from '../../services/toast.service';
import {SpinnerComponent} from '../spinner/spinner.component';
import {
  isLiveRange,
  paginatesForward,
  TimeseriesEntry,
  TimeseriesExporter,
  TimeseriesSource,
  TimeseriesSourceWithStatus,
} from './timeseries-source';

@Component({
  selector: 'app-timeseries-table',
  templateUrl: './timeseries-table.component.html',
  imports: [
    DatePipe,
    AsyncPipe,
    NgClass,
    NgTemplateOutlet,
    FaIconComponent,
    IntersectionObserverDirective,
    SpinnerComponent,
  ],
})
export class TimeseriesTableComponent {
  public readonly source = input.required<TimeseriesSource>();
  public readonly exporter = input<TimeseriesExporter>();
  public readonly resourceColorMap = input<Record<string, string>>({});
  protected readonly showResourceColumn = computed(() => Object.keys(this.resourceColorMap()).length > 1);

  protected readonly sourceWithStatus = computed(() => new TimeseriesSourceWithStatus(this.source()));
  // Derived from the source so they cannot disagree with the range it actually reads.
  private readonly range = computed(() => this.sourceWithStatus().range);
  protected readonly newestFirst = computed(() => this.range().order === 'DESC');
  protected readonly live = computed(() => isLiveRange(this.range()));
  protected readonly paginateForward = computed(() => paginatesForward(this.range()));

  private readonly toastService = inject(ToastService);
  private readonly injector = inject(Injector);

  protected readonly faArrowDown = faArrowDown;
  protected readonly faThumbtack = faThumbtack;
  protected readonly faThumbtackSlash = faThumbtackSlash;

  private static readonly LIVE_INTERVAL_MS = 10_000;

  protected readonly hasMore = signal(true);
  protected readonly pinnedEntryId = signal<string | null>(null);
  protected readonly userIsAtBottom = signal(true);
  protected readonly showScrollToBottom = computed(() => !this.userIsAtBottom() && this.live() && !this.newestFirst());
  private readonly liveResetTimestamp = signal(Date.now());

  protected readonly liveProgress = toSignal(
    combineLatest([toObservable(this.live), toObservable(this.liveResetTimestamp)]).pipe(
      switchMap(([live, resetTimestamp]) =>
        live
          ? interval(100).pipe(
              map(() =>
                Math.min(100, ((Date.now() - resetTimestamp) / TimeseriesTableComponent.LIVE_INTERVAL_MS) * 100)
              )
            )
          : EMPTY
      )
    ),
    {initialValue: 0}
  );

  protected readonly entries$: Observable<TimeseriesEntry[]> = toObservable(this.sourceWithStatus).pipe(
    switchMap((source) => {
      const newestFirst = source.range.order === 'DESC';
      const live = isLiveRange(source.range);
      const paginateForward = paginatesForward(source.range);
      let nextBefore: Date | null = null;
      let nextAfter: Date | null = null;
      return merge(
        merge(
          source.load().pipe(catchError((err) => this.handleError(err))),
          this.loadMore$.pipe(
            exhaustMap(() => {
              if (paginateForward) {
                return nextAfter !== null
                  ? source.loadAfter(nextAfter).pipe(catchError((err) => this.handleError(err)))
                  : EMPTY;
              } else {
                return nextBefore !== null
                  ? source.loadBefore(nextBefore).pipe(catchError((err) => this.handleError(err)))
                  : EMPTY;
              }
            })
          )
        ).pipe(tap((entries) => this.hasMore.set(entries.length >= source.batchSize))),
        live
          ? interval(TimeseriesTableComponent.LIVE_INTERVAL_MS).pipe(
              switchMap(() =>
                nextAfter !== null
                  ? source.loadAfter(nextAfter).pipe(catchError((err) => this.handleError(err)))
                  : EMPTY
              ),
              tap((entries) => {
                this.liveResetTimestamp.set(Date.now());
                if (!newestFirst && entries.length > 0) {
                  this.autoScrollIfAtBottom();
                }
              })
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
        ),
        map((entries) => entries.slice().sort(compareByDate(newestFirst)))
      );
    })
  );

  private readonly loadMore$ = new Subject<void>();
  private readonly tableBottom = viewChild<ElementRef<HTMLElement>>('tableBottom');

  protected loadMore() {
    this.loadMore$.next();
  }

  protected onLoadMoreVisible(isIntersecting: boolean) {
    if (isIntersecting) {
      this.loadMore();
    }
  }

  private autoScrollIfAtBottom() {
    const bottomEl = this.tableBottom()?.nativeElement;
    const isAtBottom = !bottomEl || bottomEl.getBoundingClientRect().top < window.innerHeight + 100;
    this.userIsAtBottom.set(isAtBottom);
    if (isAtBottom) {
      afterNextRender(() => this.scrollToBottom(), {injector: this.injector});
    }
  }

  protected scrollToBottom() {
    this.userIsAtBottom.set(true);
    this.tableBottom()?.nativeElement.scrollIntoView({behavior: 'smooth'});
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

    const today = new Date().toISOString().split('T')[0];
    const filename = `${today}_${exporter.getFileName()}`;
    const toastRef = this.toastService.info('Download started...');

    exporter.export().subscribe({
      next: (blob) => {
        downloadBlob(blob, filename);
        toastRef?.close();
        this.toastService.success('Download completed successfully');
      },
      error: (err) => {
        console.error('Export failed:', err);
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
