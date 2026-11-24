import {inject} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FormControl} from '@angular/forms';
import {ActivatedRoute, Router} from '@angular/router';
import {combineLatest, debounceTime, distinctUntilChanged, map, Observable, startWith} from 'rxjs';

export function filteredByFormControl<T>(
  dataSource: Observable<T[]>,
  formControl: FormControl<string | null>,
  matchFn: (item: T, search: string) => boolean
): Observable<T[]> {
  return combineLatest([dataSource, formControl.valueChanges.pipe(startWith(formControl.value))]).pipe(
    map(([items, search]) => {
      return items.filter((it) => !search || matchFn(it, search));
    })
  );
}

/** Keeps the search control in sync with the `search` query parameter. Call it in an injection context. */
export function syncSearchWithQueryParam(search: FormControl<string | null>) {
  const route = inject(ActivatedRoute);
  const router = inject(Router);
  // The component is reused when only the query changes, e.g. when another page links to the one it is on.
  route.queryParamMap
    .pipe(
      map((params) => params.get('search') ?? ''),
      distinctUntilChanged(),
      takeUntilDestroyed()
    )
    .subscribe((value) => {
      if (value !== search.value) {
        search.setValue(value);
      }
    });
  search.valueChanges.pipe(debounceTime(300), takeUntilDestroyed()).subscribe((value) =>
    router.navigate([], {
      relativeTo: route,
      queryParams: {search: value || null},
      queryParamsHandling: 'merge',
      replaceUrl: true,
    })
  );
}
