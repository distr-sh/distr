import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map} from 'rxjs';
import {AuthService} from '../../services/auth.service';
import {OrganizationService} from '../../services/organization.service';
import {AlertComponent} from '../alert/alert.component';
import {PageComponent} from '../page.component';

@Component({
  selector: 'app-portal-unavailable',
  templateUrl: './portal-unavailable.component.html',
  imports: [PageComponent, AlertComponent],
})
export class PortalUnavailableComponent {
  private readonly organizationService = inject(OrganizationService);
  protected readonly isVendor = inject(AuthService).isVendor();
  protected readonly vendorName = toSignal(this.organizationService.get().pipe(map((org) => org.name)));
}
