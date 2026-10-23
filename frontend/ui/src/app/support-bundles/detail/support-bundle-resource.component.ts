import {Component, inject, input, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronRight, faSpinner} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {BytesPipe} from '../../../util/units';
import {SupportBundlesService} from '../../services/support-bundles.service';
import {ToastService} from '../../services/toast.service';
import {SupportBundleResourceSummary} from '../../types/support-bundle';

@Component({
  selector: 'app-support-bundle-resource',
  templateUrl: './support-bundle-resource.component.html',
  imports: [FaIconComponent, BytesPipe],
  host: {class: 'block'},
})
export class SupportBundleResourceComponent {
  private readonly supportBundlesService = inject(SupportBundlesService);
  private readonly toast = inject(ToastService);

  public readonly bundleId = input.required<string>();
  public readonly resource = input.required<SupportBundleResourceSummary>();

  protected readonly faChevronRight = faChevronRight;
  protected readonly faSpinner = faSpinner;

  protected readonly content = signal<string | undefined>(undefined);
  protected readonly loading = signal(false);

  protected async loadContent(open: boolean): Promise<void> {
    if (!open || this.content() !== undefined || this.loading()) {
      return;
    }
    this.loading.set(true);
    try {
      const resource = await firstValueFrom(
        this.supportBundlesService.getResource(this.bundleId(), this.resource().id)
      );
      this.content.set(resource.content);
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    } finally {
      this.loading.set(false);
    }
  }
}
