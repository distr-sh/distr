import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {firstValueFrom, Observable} from 'rxjs';
import {CheckoutRequest, SubscriptionInfo, UpdateSubscriptionRequest} from '../types/subscription';

@Injectable({
  providedIn: 'root',
})
export class SubscriptionService {
  private readonly httpClient = inject(HttpClient);
  private readonly baseUrl = '/api/v1/billing';

  get(): Observable<SubscriptionInfo> {
    return this.httpClient.get<SubscriptionInfo>(`${this.baseUrl}/subscription`);
  }

  async checkout(request: CheckoutRequest): Promise<void> {
    // Create checkout session on backend
    const response = await firstValueFrom(
      this.httpClient.post<{
        url: string;
      }>(`${this.baseUrl}/subscription`, request)
    );

    window.location.href = response.url;
  }

  async updateSubscription(request: UpdateSubscriptionRequest): Promise<SubscriptionInfo> {
    // Update subscription quantities and optionally switch the plan
    const response = await firstValueFrom(
      this.httpClient.put<SubscriptionInfo>(`${this.baseUrl}/subscription`, request)
    );

    if (!response) {
      throw new Error('Failed to update subscription');
    }

    return response;
  }

  async openBillingPortal(returnUrl?: string): Promise<void> {
    // Create billing portal session on backend
    const response = await firstValueFrom(
      this.httpClient.post<{
        url: string;
      }>(`${this.baseUrl}/portal`, {
        returnUrl: returnUrl || window.location.href,
      })
    );

    if (!response?.url) {
      throw new Error('Failed to create billing portal session');
    }

    window.location.href = response.url;
  }
}
