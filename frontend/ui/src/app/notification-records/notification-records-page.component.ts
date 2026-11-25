import {Component} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft} from '@fortawesome/free-solid-svg-icons';
import {PageComponent} from '../components/page.component';
import {NotificationRecordsComponent} from './notification-records.component';

@Component({
  template: `
    <app-page>
      <a routerLink="/notifications" class="distr-link mb-4 inline-flex items-center text-sm">
        <fa-icon [icon]="faArrowLeft" class="mr-2" />
        Notifications
      </a>
      <div class="distr-table-panel">
        <app-notification-records />
      </div>
    </app-page>
  `,
  imports: [PageComponent, NotificationRecordsComponent, RouterLink, FaIconComponent],
})
export class NotificationRecordsPageComponent {
  protected readonly faArrowLeft = faArrowLeft;
}
