import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {startWith, Subject, switchMap} from 'rxjs';
import {AuthService} from '../../../services/auth.service';
import {UsersService} from '../../../services/users.service';
import {UsersComponent} from '../users.component';

@Component({
  template: `<section class="bg-gray-50 dark:bg-gray-900 p-3 sm:p-5 antialiased">
    <div class="mx-auto max-w-screen-2xl px-4 lg:px-12">
      <div class="bg-white dark:bg-gray-800 relative shadow-md sm:rounded-lg overflow-hidden">
        <app-users (refresh)="refresh$.next()" [users]="users()" />
      </div>
    </div>
  </section>`,
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [UsersComponent],
})
export class VendorUsersComponent {
  private readonly usersService = inject(UsersService);
  private readonly auth = inject(AuthService);
  protected readonly refresh$ = new Subject<void>();

  private readonly allUsers = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.usersService.getUsers())
    )
  );

  protected readonly users = computed(() => {
    const all = this.allUsers() ?? [];
    if (this.auth.isVendor()) {
      return all.filter(
        (user) => user.customerOrganizationId === undefined && user.partnerOrganizationId === undefined
      );
    } else if (this.auth.isPartner()) {
      const partnerOrgId = this.auth.getPartnerOrganizationId();
      return all.filter((user) => user.partnerOrganizationId === partnerOrgId);
    }
    return all;
  });
}
