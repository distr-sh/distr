import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {combineLatest, map, startWith, Subject, switchMap} from 'rxjs';
import {UsersService} from '../../../services/users.service';
import {CustomerBreadcrumbComponent} from '../../customer-breadcrumb.component';
import {PageComponent} from '../../page.component';
import {UsersComponent} from '../users.component';

@Component({
  templateUrl: './customer-users.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [UsersComponent, CustomerBreadcrumbComponent, PageComponent],
})
export class CustomerUsersComponent {
  private readonly usersService = inject(UsersService);
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );

  protected readonly refresh$ = new Subject<void>();
  protected readonly users = toSignal(
    combineLatest([
      this.refresh$.pipe(
        startWith(undefined),
        switchMap(() => this.usersService.getUsers())
      ),
      toObservable(this.customerOrganizationId),
    ]).pipe(
      map(([users, customerOrganizationId]) =>
        users.filter((it) => it.customerOrganizationId === customerOrganizationId)
      )
    )
  );
}
