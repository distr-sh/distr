import {HttpClient} from '@angular/common/http';
import {inject, Injectable} from '@angular/core';
import {Router} from '@angular/router';
import {firstValueFrom} from 'rxjs';
import {ToastService} from './toast.service';

const REDIRECT_URL_STORAGE_KEY = 'maintenance.redirectUrl';

@Injectable({providedIn: 'root'})
export class MaintenanceService {
  private readonly http = inject(HttpClient);
  private readonly router = inject(Router);
  private readonly toast = inject(ToastService);

  private inMaintenance = false;
  private checking = false;

  async checkReady(): Promise<boolean> {
    try {
      const response = await firstValueFrom(this.http.get('/ready', {observe: 'response'}));
      return response.status === 200;
    } catch {
      return false;
    }
  }

  async handleServerError(): Promise<void> {
    if (this.inMaintenance || this.checking) {
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
    if (this.inMaintenance) {
      return;
    }
    this.inMaintenance = true;
    const url = this.router.url;
    if (!url.startsWith('/maintenance')) {
      // Persisted so that a full browser reload while on the maintenance page can still recover.
      sessionStorage.setItem(REDIRECT_URL_STORAGE_KEY, url);
    }
    this.router.navigateByUrl('/maintenance');
  }

  recover(): void {
    this.inMaintenance = false;
    const target = sessionStorage.getItem(REDIRECT_URL_STORAGE_KEY) ?? '/';
    sessionStorage.removeItem(REDIRECT_URL_STORAGE_KEY);
    this.router.navigateByUrl(target);
  }
}
