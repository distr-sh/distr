import {Component} from '@angular/core';
import {PageComponent} from '../components/page.component';
import {LicenseKeysComponent} from './license-keys/license-keys.component';

@Component({
  selector: 'app-customer-license-detail-page',
  imports: [LicenseKeysComponent, PageComponent],
  template: `
    <app-page>
      <app-license-keys />
    </app-page>
  `,
})
export class CustomerLicenseDetailPageComponent {}
