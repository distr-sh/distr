import {Component, input} from '@angular/core';
import {DeploymentType} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDocker} from '@fortawesome/free-brands-svg-icons';
import {faDharmachakra} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-deployment-type-badge',
  imports: [FaIconComponent],
  host: {class: 'distr-badge'},
  template: `
    @switch (type()) {
      @case ('docker') {
        <fa-icon [icon]="faDocker" />
        Docker
      }
      @case ('kubernetes') {
        <fa-icon [icon]="faDharmachakra" />
        Kubernetes
      }
    }
  `,
})
export class DeploymentTypeBadgeComponent {
  public readonly type = input.required<DeploymentType>();

  protected readonly faDocker = faDocker;
  protected readonly faDharmachakra = faDharmachakra;
}
