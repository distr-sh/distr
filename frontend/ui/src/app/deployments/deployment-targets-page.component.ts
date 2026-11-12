import {Component} from '@angular/core';
import {PageComponent} from '../components/page.component';
import {DeploymentTargetsComponent} from './deployment-targets.component';

@Component({
  template: `
    <app-page>
      <app-deployment-targets />
    </app-page>
  `,
  imports: [PageComponent, DeploymentTargetsComponent],
})
export class DeploymentTargetsPageComponent {}
