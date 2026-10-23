import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {ActivityFeedEntry} from '../types/activity-feed';

@Injectable({providedIn: 'root'})
export class ActivityFeedService {
  private readonly httpClient = inject(HttpClient);

  public list(): Observable<ActivityFeedEntry[]> {
    return this.httpClient.get<ActivityFeedEntry[]>('/api/v1/activity-feed');
  }
}
