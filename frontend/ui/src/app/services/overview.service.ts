import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Observable} from 'rxjs';
import {
  OverviewAccessToken,
  OverviewActivity,
  OverviewAlerts,
  OverviewArtifacts,
  OverviewChecklist,
  OverviewDeploymentTarget,
  OverviewLicense,
  OverviewSupportBundle,
} from '../types/overview';

@Injectable({providedIn: 'root'})
export class OverviewService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = '/api/v1/overview';

  getDeploymentTargets(): Observable<OverviewDeploymentTarget[]> {
    return this.httpClient.get<OverviewDeploymentTarget[]>(`${this.baseUrl}/deployment-targets`);
  }

  getArtifacts(): Observable<OverviewArtifacts> {
    return this.httpClient.get<OverviewArtifacts>(`${this.baseUrl}/artifacts`);
  }

  getChecklist(): Observable<OverviewChecklist> {
    return this.httpClient.get<OverviewChecklist>(`${this.baseUrl}/checklist`);
  }

  /** Emits null when the customer has neither a license key nor an entitlement. */
  getLicense(): Observable<OverviewLicense | null> {
    return this.httpClient.get<OverviewLicense | null>(`${this.baseUrl}/license`);
  }

  getAlerts(): Observable<OverviewAlerts> {
    return this.httpClient.get<OverviewAlerts>(`${this.baseUrl}/alerts`);
  }

  getAccessTokens(): Observable<OverviewAccessToken[]> {
    return this.httpClient.get<OverviewAccessToken[]>(`${this.baseUrl}/access-tokens`);
  }

  getSupportBundles(): Observable<OverviewSupportBundle[]> {
    return this.httpClient.get<OverviewSupportBundle[]>(`${this.baseUrl}/support-bundles`);
  }

  getActivity(): Observable<OverviewActivity[]> {
    return this.httpClient.get<OverviewActivity[]>(`${this.baseUrl}/activity`);
  }
}
