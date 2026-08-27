import {signal} from '@angular/core';
import {map, Observable, tap} from 'rxjs';
import {OrderDirection, TimeseriesOptions} from '../../types/timeseries-options';

export interface TimeseriesEntry {
  id?: string;
  date: string;
  status: string;
  detail: string;
  resource?: string;
}

/** The time range, order and body filter a source reads with. */
export interface TimeseriesRange {
  order: OrderDirection;
  after?: Date;
  before?: Date;
  filter?: string;
}

export interface TimeseriesSource {
  readonly batchSize: number;
  readonly range: TimeseriesRange;
  load(): Observable<TimeseriesEntry[]>;
  loadBefore(before: Date): Observable<TimeseriesEntry[]>;
  loadAfter(after: Date): Observable<TimeseriesEntry[]>;
}

export function isLiveRange(range: TimeseriesRange): boolean {
  return !range.after && !range.before;
}

/**
 * Whether paging continues towards newer entries. This must match the order the API
 * returns the first page in, which is only oldest-first when an "after" bound is set.
 */
export function paginatesForward(range: TimeseriesRange): boolean {
  return range.order === 'ASC' && !!range.after;
}

/**
 * A source that reads a fixed range through `fetch` and maps each record with `toEntry`.
 */
export class RangeTimeseriesSource<T> implements TimeseriesSource {
  public readonly batchSize = 50;

  constructor(
    private readonly fetch: (options: TimeseriesOptions) => Observable<T[]>,
    private readonly toEntry: (record: T) => TimeseriesEntry,
    public readonly range: TimeseriesRange
  ) {}

  load(): Observable<TimeseriesEntry[]> {
    return this.query({});
  }

  loadAfter(after: Date): Observable<TimeseriesEntry[]> {
    return this.query({after});
  }

  loadBefore(before: Date): Observable<TimeseriesEntry[]> {
    return this.query({before});
  }

  // A cursor narrows the configured range and never replaces it, so paging cannot leave
  // the range the user selected.
  private query(cursor: {after?: Date; before?: Date}): Observable<TimeseriesEntry[]> {
    return this.fetch({
      limit: this.batchSize,
      after: cursor.after ?? this.range.after,
      before: cursor.before ?? this.range.before,
      filter: this.range.filter,
      order: this.range.order,
    }).pipe(map((records) => records.map((record) => this.toEntry(record))));
  }
}

export class TimeseriesSourceWithStatus implements TimeseriesSource {
  public readonly batchSize: number;
  public readonly range: TimeseriesRange;
  private readonly loadingRW = signal(false);
  public readonly loading = this.loadingRW.asReadonly();

  constructor(private readonly source: TimeseriesSource) {
    this.batchSize = source.batchSize;
    this.range = source.range;
  }

  load(): Observable<TimeseriesEntry[]> {
    return this.withStatus(this.source.load());
  }

  loadBefore(before: Date): Observable<TimeseriesEntry[]> {
    return this.withStatus(this.source.loadBefore(before));
  }

  loadAfter(after: Date): Observable<TimeseriesEntry[]> {
    return this.withStatus(this.source.loadAfter(after));
  }

  private withStatus(entries: Observable<TimeseriesEntry[]>): Observable<TimeseriesEntry[]> {
    this.loadingRW.set(true);
    return entries.pipe(tap({finalize: () => this.loadingRW.set(false)}));
  }
}

export interface TimeseriesExporter {
  getFileName(): string;
  export(): Observable<Blob>;
}
