import {DatePipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, inject, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGear} from '@fortawesome/free-solid-svg-icons';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {SupportBundle} from '../../types/support-bundle';

@Component({
  selector: 'app-vendor-support-bundle-list',
  templateUrl: './vendor-support-bundle-list.component.html',
  imports: [RouterLink, FaIconComponent, DatePipe],
})
export class VendorSupportBundleListComponent {
  protected readonly faGear = faGear;

  private readonly svc = inject(SupportBundlesService);

  protected readonly loading = signal(true);
  protected readonly configExists = signal(false);
  protected readonly bundles = signal<SupportBundle[]>([]);

  constructor() {
    this.svc
      .getConfiguration()
      .pipe(takeUntilDestroyed())
      .subscribe({
        next: () => this.configExists.set(true),
        error: (e) => {
          if (e instanceof HttpErrorResponse && e.status === 404) {
            this.configExists.set(false);
          }
        },
      });

    this.svc
      .list()
      .pipe(takeUntilDestroyed())
      .subscribe({
        next: (bundles) => {
          this.bundles.set(bundles);
          this.loading.set(false);
        },
        error: () => {
          this.loading.set(false);
        },
      });
  }

  protected statusBadgeClass(status: SupportBundle['status']): string {
    switch (status) {
      case 'initialized':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300';
      case 'created':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300';
      case 'resolved':
        return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300';
      case 'canceled':
        return 'bg-gray-100 text-gray-800 dark:bg-gray-900 dark:text-gray-300';
      default:
        return '';
    }
  }

  protected statusLabel(status: SupportBundle['status']): string {
    return status.charAt(0).toUpperCase() + status.slice(1);
  }
}
