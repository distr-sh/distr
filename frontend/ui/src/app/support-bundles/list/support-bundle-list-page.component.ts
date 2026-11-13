import {Component} from '@angular/core';
import {PageComponent} from '../../components/page.component';
import {SupportBundleListComponent} from './support-bundle-list.component';

@Component({
  template: `
    <app-page>
      <app-support-bundle-list />
    </app-page>
  `,
  imports: [PageComponent, SupportBundleListComponent],
})
export class SupportBundleListPageComponent {}
