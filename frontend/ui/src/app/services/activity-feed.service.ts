import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {ActivityFeedEntry} from '../types/activity-feed';
import {customerScopeParams} from './customer-scope';

@Injectable({providedIn: 'root'})
export class ActivityFeedService {
  private readonly httpClient = inject(HttpClient);

  /** A vendor or partner has to pass the customer organization. */
  public list(customerOrganizationId?: string): Observable<ActivityFeedEntry[]> {
    return this.httpClient.get<ActivityFeedEntry[]>('/api/v1/activity-feed', {
      params: customerScopeParams(customerOrganizationId),
    });
  }
}
