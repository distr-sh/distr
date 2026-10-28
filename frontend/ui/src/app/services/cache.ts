import {BaseModel, Named} from '@distr-sh/distr-sdk';
import {BehaviorSubject, defer, filter, firstValueFrom, Observable} from 'rxjs';
import {compareBy, distinctBy, Predicate} from '../../util/arrays';

export abstract class ReactiveList<T> {
  protected abstract readonly identify: Predicate<T, unknown>;
  protected abstract readonly sortAttr: Predicate<T, string>;
  protected readonly sortInverted: boolean = false;

  private readonly state$ = new BehaviorSubject<T[] | undefined>(undefined);
  private requested = false;
  private inFlight?: Promise<T[]>;

  private readonly items$ = defer(() => {
    if (!this.requested) {
      // A failed first load leaves the list empty until the next refresh, and whoever awaits refresh() reports it.
      this.refresh().catch(() => undefined);
    }
    return this.state$;
  }).pipe(filter((items): items is T[] => items !== undefined));

  constructor(private readonly fetch: () => Observable<T[]>) {}

  /** Concurrent calls share one request, so callers never need to guard against refreshing twice. */
  public refresh(): Promise<T[]> {
    this.requested = true;
    this.inFlight ??= firstValueFrom(this.fetch())
      .then((items) => {
        this.reset(items);
        return this.state$.value!;
      })
      .catch((e) => {
        if (this.state$.value === undefined) {
          this.requested = false;
        }
        throw e;
      })
      .finally(() => (this.inFlight = undefined));
    return this.inFlight;
  }

  public save(arg: T) {
    this.next(distinctBy(this.identify)([arg, ...(this.state$.value ?? [])]));
  }

  public remove(arg: T) {
    const state = this.state$.value;
    if (state !== undefined) {
      this.next(state.filter((item) => this.identify(item) !== this.identify(arg)));
    }
  }

  public reset(arg: T[]) {
    this.next(arg);
  }

  public get(): Observable<T[]> {
    return this.items$;
  }

  private next(items: T[]) {
    this.state$.next([...items].sort(compareBy(this.sortAttr, this.sortInverted)));
  }
}

export class DefaultReactiveList<T extends Named & BaseModel> extends ReactiveList<T> {
  protected override identify = (it: T) => it.id;
  protected override sortAttr = (it: T) => it.name!;
}
