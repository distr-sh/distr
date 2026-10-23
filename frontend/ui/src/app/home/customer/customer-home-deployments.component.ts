import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {NgTemplateOutlet} from '@angular/common';
import {Component, inject, input, output, TemplateRef, viewChild} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBox, faDashboard, faDownload, faPlus} from '@fortawesome/free-solid-svg-icons';
import {DeploymentWizardComponent} from '../../deployments/deployment-wizard/deployment-wizard.component';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {CustomerHomeArtifacts, CustomerHomeDeploymentTarget} from '../../types/customer-home';
import {CustomerHomeArtifactItemComponent} from './customer-home-artifact-item.component';
import {CustomerHomeDeploymentTargetItemComponent} from './customer-home-deployment-target-item.component';
import {CustomerHomeMenuComponent} from './customer-home-menu.component';

export type CustomerHomePrimarySection = 'deployments' | 'artifacts' | 'none';

@Component({
  selector: 'app-customer-home-deployments',
  templateUrl: './customer-home-deployments.component.html',
  imports: [
    NgTemplateOutlet,
    RouterLink,
    FaIconComponent,
    DeploymentWizardComponent,
    CustomerHomeArtifactItemComponent,
    CustomerHomeDeploymentTargetItemComponent,
    CustomerHomeMenuComponent,
  ],
})
export class CustomerHomeDeploymentsComponent {
  private readonly overlay = inject(OverlayService);

  public readonly section = input.required<CustomerHomePrimarySection>();
  public readonly deploymentTargets = input.required<CustomerHomeDeploymentTarget[]>();
  public readonly artifacts = input.required<CustomerHomeArtifacts>();
  public readonly instructionsHidden = input.required<boolean>();
  public readonly deploymentCreated = output<void>();
  public readonly showInstructions = output<void>();

  protected readonly faBox = faBox;
  protected readonly faDashboard = faDashboard;
  protected readonly faDownload = faDownload;
  protected readonly faPlus = faPlus;

  private readonly deploymentWizard = viewChild.required<TemplateRef<unknown>>('deploymentWizard');
  private modal?: DialogRef;

  protected openWizard() {
    this.modal?.close();
    this.modal = this.overlay.showModal(this.deploymentWizard(), {
      hasBackdrop: true,
      backdropStyleOnly: true,
      positionStrategy: new GlobalPositionStrategy().centerHorizontally().centerVertically(),
    });
  }

  protected closeWizard() {
    this.modal?.close();
    this.modal = undefined;
    this.deploymentCreated.emit();
  }
}
