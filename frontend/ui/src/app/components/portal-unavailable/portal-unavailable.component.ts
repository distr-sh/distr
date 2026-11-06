import {Component} from '@angular/core';
import {AlertComponent} from '../alert/alert.component';
import {PageComponent} from '../page.component';

@Component({
  selector: 'app-portal-unavailable',
  templateUrl: './portal-unavailable.component.html',
  imports: [PageComponent, AlertComponent],
})
export class PortalUnavailableComponent {}
