import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map} from 'rxjs';
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

  protected readonly users = toSignal(
    this.usersService.getUsers().pipe(map((users) => users.filter((user) => organizationKind(user) === 'vendor')))
  );

  constructor() {
    refreshInBackground(this.usersService.refresh());
  }
}
