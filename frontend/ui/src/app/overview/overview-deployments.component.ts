import {GlobalPositionStrategy} from '@angular/cdk/overlay';
import {NgTemplateOutlet} from '@angular/common';
import {Component, inject, input, output, TemplateRef, viewChild} from '@angular/core';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faBox, faDashboard, faDownload, faPlus} from '@fortawesome/free-solid-svg-icons';
import {DeploymentWizardComponent} from '../deployments/deployment-wizard/deployment-wizard.component';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {OverviewArtifacts, OverviewDeploymentTarget} from '../types/overview';
import {OverviewArtifactItemComponent} from './overview-artifact-item.component';
import {OverviewDeploymentTargetItemComponent} from './overview-deployment-target-item.component';
import {OverviewMenuComponent} from './overview-menu.component';

export type OverviewPrimarySection = 'deployments' | 'artifacts' | 'none';

@Component({
  selector: 'app-overview-deployments',
  templateUrl: './overview-deployments.component.html',
  imports: [
    NgTemplateOutlet,
    RouterLink,
    FaIconComponent,
    DeploymentWizardComponent,
    OverviewArtifactItemComponent,
    OverviewDeploymentTargetItemComponent,
    OverviewMenuComponent,
  ],
})
export class OverviewDeploymentsComponent {
  private readonly overlay = inject(OverlayService);

  public readonly section = input.required<OverviewPrimarySection>();
  public readonly deploymentTargets = input.required<OverviewDeploymentTarget[]>();
  public readonly artifacts = input.required<OverviewArtifacts>();
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
