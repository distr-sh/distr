import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map, startWith, Subject, switchMap} from 'rxjs';
import {ServiceAccountsComponent} from '../../../service-accounts/service-accounts.component';
import {AuthService} from '../../../services/auth.service';
import {UsersService} from '../../../services/users.service';
import {UsersComponent} from '../users.component';

@Component({
  template: `<section class="bg-gray-50 dark:bg-gray-900 p-3 sm:p-5 antialiased">
      <div class="mx-auto max-w-screen-2xl px-4 lg:px-12">
        <div class="bg-white dark:bg-gray-800 relative shadow-md sm:rounded-lg overflow-hidden">
          <app-users (refresh)="refresh$.next()" [users]="users() ?? []" />
        </div>
      </div>
    </section>
    @if (auth.hasRole('admin')) {
      <app-service-accounts />
    }`,
  imports: [UsersComponent, ServiceAccountsComponent],
})
export class VendorUsersComponent {
  private readonly usersService = inject(UsersService);
  protected readonly auth = inject(AuthService);
  protected readonly refresh$ = new Subject<void>();
  protected readonly users = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.usersService.getUsers()),
      map((users) => (this.auth.isVendor() ? users.filter((user) => user.customerOrganizationId === undefined) : users))
    )
  );
}
