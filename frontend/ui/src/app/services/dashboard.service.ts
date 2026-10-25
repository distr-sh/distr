import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {CustomerOrganization} from '@distr-sh/distr-sdk';
import {Observable} from 'rxjs';
import {ArtifactWithTags} from './artifacts.service';

export interface DashboardArtifact {
  artifact: ArtifactWithTags;
  latestPulledVersion: string;
}

export interface ArtifactsByCustomer {
  customer: CustomerOrganization;
  artifacts?: DashboardArtifact[];
}

@Injectable({providedIn: 'root'})
export class DashboardService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = '/api/v1/dashboard';

  public getArtifactsByCustomer(): Observable<ArtifactsByCustomer[]> {
    return this.httpClient.get<ArtifactsByCustomer[]>(`${this.baseUrl}/artifacts-by-customer`);
  }
}
