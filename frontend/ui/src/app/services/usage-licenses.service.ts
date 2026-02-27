import {HttpClient} from '@angular/common/http';
import {Injectable} from '@angular/core';
import {Observable, Subject, switchMap, tap} from 'rxjs';
import {UsageLicense} from '../types/usage-license';
import {DefaultReactiveList, ReactiveList} from './cache';
import {CrudService} from './interfaces';

@Injectable({providedIn: 'root'})
export class UsageLicensesService implements CrudService<UsageLicense> {
  private readonly cache: ReactiveList<UsageLicense>;
  private readonly usageLicensesUrl = '/api/v1/usage-licenses';
  private readonly refresh$ = new Subject<void>();

  constructor(private readonly http: HttpClient) {
    this.cache = new DefaultReactiveList(this.http.get<UsageLicense[]>(this.usageLicensesUrl));
    this.refresh$
      .pipe(
        switchMap(() => this.http.get<UsageLicense[]>(this.usageLicensesUrl)),
        tap((licenses) => this.cache.reset(licenses))
      )
      .subscribe();
  }

  public list(): Observable<UsageLicense[]> {
    return this.cache.get();
  }

  refresh() {
    this.refresh$.next();
  }

  create(request: UsageLicense): Observable<UsageLicense> {
    return this.http.post<UsageLicense>(this.usageLicensesUrl, request).pipe(tap((l) => this.cache.save(l)));
  }

  update(request: UsageLicense): Observable<UsageLicense> {
    return this.http
      .put<UsageLicense>(`${this.usageLicensesUrl}/${request.id}`, request)
      .pipe(tap((l) => this.cache.save(l)));
  }

  delete(request: UsageLicense): Observable<void> {
    return this.http.delete<void>(`${this.usageLicensesUrl}/${request.id}`).pipe(tap(() => this.cache.remove(request)));
  }
}
