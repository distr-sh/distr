import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {ControllerVersion} from '@distr-sh/distr-sdk';
import {Observable, shareReplay} from 'rxjs';

const baseUrl = '/api/v1/controller-versions';

@Injectable({providedIn: 'root'})
export class ControllerVersionService {
  private readonly httpClient = inject(HttpClient);
  private readonly controllerVersions$ = this.httpClient.get<ControllerVersion[]>(baseUrl).pipe(shareReplay(1));

  public list(): Observable<ControllerVersion[]> {
    return this.controllerVersions$;
  }
}
