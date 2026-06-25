import {ChangeDetectionStrategy, Component, inject, signal} from '@angular/core';
import {takeUntilDestroyed} from '@angular/core/rxjs-interop';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faScrewdriverWrench} from '@fortawesome/free-solid-svg-icons';
import {timer} from 'rxjs';
import {WEBSITE_URL} from '../../constants';
import {MaintenanceService} from '../services/maintenance.service';

const POLL_INTERVAL_MS = 5000;

@Component({
  selector: 'app-maintenance',
  imports: [FaIconComponent],
  changeDetection: ChangeDetectionStrategy.OnPush,
  templateUrl: './maintenance.component.html',
})
export class MaintenanceComponent {
  private readonly maintenance = inject(MaintenanceService);

  protected readonly websiteUrl = WEBSITE_URL;
  protected readonly faScrewdriverWrench = faScrewdriverWrench;
  protected readonly checking = signal(false);

  constructor() {
    timer(0, POLL_INTERVAL_MS)
      .pipe(takeUntilDestroyed())
      .subscribe(() => this.check());
  }

  protected async check(): Promise<void> {
    if (this.checking()) {
      return;
    }
    this.checking.set(true);
    setTimeout(async () => {
      try {
        if (await this.maintenance.checkReady()) {
          this.maintenance.recover();
        }
      } finally {
        this.checking.set(false);
      }
    }, 500);
  }
}
