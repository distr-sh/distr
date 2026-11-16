import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {SidebarLinksComponent} from '../../../sidebar-links/sidebar-links.component';

@Component({
  templateUrl: './customer-links-tab.component.html',
  imports: [SidebarLinksComponent],
})
export class CustomerLinksTabComponent {
  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );
}
