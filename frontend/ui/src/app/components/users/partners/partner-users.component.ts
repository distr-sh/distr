import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {refreshInBackground} from '../../../../util/refresh';
import {PartnerOrganizationsService} from '../../../services/partner-organizations.service';
import {UsersService} from '../../../services/users.service';
import {PageComponent} from '../../page.component';
import {UsersComponent} from '../users.component';

@Component({
  templateUrl: './partner-users.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [UsersComponent, PageComponent],
})
export class PartnerUsersComponent {
  private readonly partnerOrganizationsService = inject(PartnerOrganizationsService);
  private readonly usersService = inject(UsersService);
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);

  protected readonly partnerOrganizationId = computed(
    () => this.routeParams()?.['partnerOrganizationId'] as string | undefined
  );

  protected readonly partnerOrganizations = toSignal(this.partnerOrganizationsService.getPartnerOrganizations());
  protected readonly partnerOrganization = computed(() => {
    const id = this.partnerOrganizationId();
    return this.partnerOrganizations()?.find((org) => org.id === id);
  });

  private readonly allUsers = toSignal(this.usersService.getUsers());
  protected readonly users = computed(() =>
    this.allUsers()?.filter((it) => it.partnerOrganizationId === this.partnerOrganizationId())
  );

  constructor() {
    refreshInBackground(this.usersService.refresh());
  }
}
