import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Router} from '@angular/router';
import {firstValueFrom, timeout} from 'rxjs';
import {ToastService} from './toast.service';

const REDIRECT_URL_STORAGE_KEY = 'maintenance.redirectUrl';
const READY_CHECK_TIMEOUT_MS = 10000;

@Injectable({providedIn: 'root'})
export class MaintenanceService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);

  private checking = false;

  async checkReady(): Promise<boolean> {
    try {
      // Bound the probe so a hanging request can never leave callers (e.g. the checking guard) stuck.
      const response = await firstValueFrom(
        this.http.get('/ready', {observe: 'response'}).pipe(timeout(READY_CHECK_TIMEOUT_MS))
      );
      return response.status === 200;
    } catch {
      return false;
    }
  }

  async handleServerError(): Promise<void> {
    if (this.checking || this.isOnMaintenancePage()) {
      return;
    }
    this.checking = true;
    try {
      if (await this.checkReady()) {
        this.toast.error('An unexpected technical error occurred');
      } else {
        this.enterMaintenance();
      }
    } finally {
      this.checking = false;
    }
  }

  private enterMaintenance(): void {
    if (this.isOnMaintenancePage()) {
      return;
    }
    // Persisted so that a full browser reload while on the maintenance page can still recover.
    sessionStorage.setItem(REDIRECT_URL_STORAGE_KEY, this.router.url);
    this.router.navigateByUrl('/maintenance');
  }

  recover(): void {
    // A probe may resolve after the user already navigated away from the maintenance page; don't yank
    // them off whatever route they opened in the meantime.
    if (!this.isOnMaintenancePage()) {
      return;
    }
    const target = sessionStorage.getItem(REDIRECT_URL_STORAGE_KEY) ?? '/';
    sessionStorage.removeItem(REDIRECT_URL_STORAGE_KEY);
    this.router.navigateByUrl(target);
  }

  private isOnMaintenancePage(): boolean {
    return this.router.url.startsWith('/maintenance');
  }
}
