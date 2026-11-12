import {Component} from '@angular/core';
import {PageComponent} from '../../components/page.component';
import {ArtifactPullsComponent} from './artifact-pulls.component';

@Component({
  template: `
    <app-page>
      <app-artifact-pulls />
    </app-page>
  `,
  imports: [PageComponent, ArtifactPullsComponent],
})
export class ArtifactPullsPageComponent {}
