import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ApplicationsComponent} from './applications.component';

@Component({
  selector: 'app-applications-page',
  standalone: true,
  imports: [ApplicationsComponent],
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './applications-page.component.html',
})
export class ApplicationsPageComponent {}
