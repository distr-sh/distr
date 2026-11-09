import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {ActivityFeedEntry} from '../types/activity-feed';
import {ReactiveList} from './cache';

class ActivityFeedReactiveList extends ReactiveList<ActivityFeedEntry> {
  protected override identify = (entry: ActivityFeedEntry) => entry.id;
  protected override sortAttr = (entry: ActivityFeedEntry) => entry.createdAt;
  protected override sortInverted = true;
}

@Injectable({providedIn: 'root'})
export class ActivityFeedService {
  private readonly httpClient = inject(HttpClient);
  private readonly cache = new ActivityFeedReactiveList(() =>
    this.httpClient.get<ActivityFeedEntry[]>('/api/v1/activity-feed')
  );

  public list(): Observable<ActivityFeedEntry[]> {
    return this.cache.get();
  }

  public refresh(): Promise<ActivityFeedEntry[]> {
    return this.cache.refresh();
  }
}
