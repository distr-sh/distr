import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map, startWith, Subject, switchMap} from 'rxjs';
import {organizationKind} from '../../../util/organization-kind';
import {refreshInBackground} from '../../../util/refresh';
import {UsersComponent} from '../../components/users/users.component';
import {UsersService} from '../../services/users.service';

@Component({
  selector: 'app-team-settings',
  templateUrl: './team-settings.component.html',
  imports: [UsersComponent],
})
export class TeamSettingsComponent {
  private readonly usersService = inject(UsersService);
  protected readonly refresh$ = new Subject<void>();

  protected readonly users = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.usersService.getUsers()),
      map((users) => users.filter((user) => organizationKind(user) === 'vendor'))
    ),
    {initialValue: []}
  );

  constructor() {
    refreshInBackground(this.usersService.refresh());
  }
}
