import {Component, inject} from '@angular/core';
import {RouterLink} from '@angular/router';
import {faCircleArrowUp} from '@fortawesome/free-solid-svg-icons';
import {CardHeadingComponent} from '../../components/card-heading.component';
import {AuthService} from '../../services/auth.service';

@Component({
  selector: 'app-customer-home-update-notifications',
  templateUrl: './customer-home-update-notifications.component.html',
  imports: [RouterLink, CardHeadingComponent],
})
export class CustomerHomeUpdateNotificationsComponent {
  protected readonly canConfigure = inject(AuthService).hasAnyRole('read_write', 'admin');
  protected readonly faCircleArrowUp = faCircleArrowUp;
}
