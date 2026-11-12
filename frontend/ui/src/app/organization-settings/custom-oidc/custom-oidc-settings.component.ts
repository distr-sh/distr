import {Component} from '@angular/core';
import {RouterLink} from '@angular/router';
import {CustomOidcComponent} from './custom-oidc.component';

@Component({
  selector: 'app-custom-oidc-settings',
  templateUrl: './custom-oidc-settings.component.html',
  imports: [CustomOidcComponent, RouterLink],
})
export class CustomOidcSettingsComponent {}
