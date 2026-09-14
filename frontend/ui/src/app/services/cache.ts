import {BaseModel, Named} from '@distr-sh/distr-sdk';
import {concat, map, Observable, scan, shareReplay, Subject} from 'rxjs';
import {compareBy, distinctBy, Predicate} from '../../util/arrays';

type ReactiveListEvent<T> = {type: 'save' | 'remove'; object: T} | {type: 'reset'; objects: T[]};

export abstract class ReactiveList<T> {
  protected abstract readonly identify: Predicate<T, unknown>;
  protected abstract readonly sortAttr: Predicate<T, string>;
  protected readonly sortInverted: boolean = false;

  private readonly events$ = new Subject<ReactiveListEvent<T>>();
  private readonly state$: Observable<T[]>;

  constructor(private readonly initial$: Observable<T[]>) {
    // TODO unhandled scenarios: initial request fails or takes too long (probably a task for the callers/components)
    this.state$ = concat(
      this.initial$.pipe(map((items) => ({type: 'reset', objects: items}) as ReactiveListEvent<T>)),
      this.events$
    ).pipe(
      scan((state: T[], event: ReactiveListEvent<T>) => {
        if (event.type === 'reset') {
          return event.objects;
        } else if (event.type === 'save') {
          return distinctBy(this.identify)([event.object, ...state]);
        } else if (event.type === 'remove') {
          return state.filter((item) => {
            return this.identify(item) !== this.identify(event.object!);
          });
        } else {
          console.warn('unsupported/invalid event: ' + event);
          return state;
        }
      }, []),
      map((ls: T[]) => [...ls].sort(compareBy(this.sortAttr, this.sortInverted))),
      shareReplay(1)
    );
  }

  public save(arg: T) {
    this.events$.next({
      type: 'save',
      object: arg,
    });
  }

  public remove(arg: T) {
    this.events$.next({
      type: 'remove',
      object: arg,
    });
  }

  public reset(arg: T[]) {
    this.events$.next({
      type: 'reset',
      objects: arg,
    });
  }

  public get(): Observable<T[]> {
    return this.state$;
  }
}

export class DefaultReactiveList<T extends Named & BaseModel> extends ReactiveList<T> {
  protected override identify = (it: T) => it.id;
  protected override sortAttr = (it: T) => it.name!;
}
