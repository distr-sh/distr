import {DatePipe, NgClass} from '@angular/common';
import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faGear} from '@fortawesome/free-solid-svg-icons';
import {map} from 'rxjs';
import {never} from '../../../util/exhaust';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {SupportBundleStatus} from '../../types/support-bundle';

@Component({
  selector: 'app-vendor-support-bundle-list',
  templateUrl: './vendor-support-bundle-list.component.html',
  imports: [RouterLink, FaIconComponent, DatePipe, NgClass],
})
export class VendorSupportBundleListComponent {
  protected readonly faGear = faGear;

  private readonly svc = inject(SupportBundlesService);

  protected readonly configExists = toSignal(this.svc.getConfiguration().pipe(map((envVars) => envVars.length > 0)), {
    initialValue: false,
  });
  private readonly bundlesResult = toSignal(this.svc.list());
  protected readonly bundles = computed(() => this.bundlesResult() ?? []);
  protected readonly loading = computed(() => this.bundlesResult() === undefined);

  protected statusBadgeClass(status: SupportBundleStatus): string {
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
        return never(status);
    }
  }
}
