import {Component} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClockRotateLeft} from '@fortawesome/free-solid-svg-icons';
import {PageComponent} from '../components/page.component';
import {NotificationsComponent} from './notifications.component';

@Component({
  templateUrl: './notifications-page.component.html',
  imports: [FaIconComponent, NotificationsComponent, PageComponent, RouterLink],
})
export class NotificationsPageComponent {
  protected readonly faClockRotateLeft = faClockRotateLeft;
}
