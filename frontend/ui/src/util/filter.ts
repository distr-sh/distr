import {FormControl} from '@angular/forms';
import {combineLatest, map, Observable, startWith} from 'rxjs';

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
