import {Component, computed, inject} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faBuilding} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, map, startWith, Subject, switchMap} from 'rxjs';
import {PartnerOrganizationsService} from '../../../services/partner-organizations.service';
import {UsersService} from '../../../services/users.service';
import {UsersComponent} from '../users.component';

@Component({
  templateUrl: './partner-users.component.html',
  imports: [UsersComponent, RouterLink, FontAwesomeModule],
})
export class PartnerUsersComponent {
  protected readonly faBuilding = faBuilding;

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

  protected readonly refresh$ = new Subject<void>();
  protected readonly users = toSignal(
    combineLatest([
      this.refresh$.pipe(
        startWith(undefined),
        switchMap(() => this.usersService.getUsers())
      ),
      toObservable(this.partnerOrganizationId),
    ]).pipe(
      map(([users, partnerOrganizationId]) => users.filter((it) => it.partnerOrganizationId === partnerOrganizationId))
    )
  );
}
