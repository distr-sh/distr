import {ChangeDetectionStrategy, Component} from '@angular/core';
import {PageComponent} from '../components/page.component';
import {ApplicationsComponent} from './applications.component';

@Component({
  selector: 'app-applications-page',
  imports: [ApplicationsComponent, PageComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './applications-page.component.html',
})
export class ApplicationsPageComponent {}
