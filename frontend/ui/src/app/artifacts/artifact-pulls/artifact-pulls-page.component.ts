import {Component} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft} from '@fortawesome/free-solid-svg-icons';
import {PageComponent} from '../../components/page.component';
import {ArtifactPullsComponent} from './artifact-pulls.component';

@Component({
  template: `
    <app-page>
      <a routerLink="/artifacts" class="distr-link mb-4 inline-flex items-center text-sm">
        <fa-icon [icon]="faArrowLeft" class="mr-2" />
        Artifacts
      </a>
      <app-artifact-pulls />
    </app-page>
  `,
  imports: [PageComponent, ArtifactPullsComponent, RouterLink, FaIconComponent],
})
export class ArtifactPullsPageComponent {
  protected readonly faArrowLeft = faArrowLeft;
}
