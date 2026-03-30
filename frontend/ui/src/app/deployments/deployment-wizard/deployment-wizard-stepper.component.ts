import {CdkStepper, CdkStepperModule} from '@angular/cdk/stepper';
import {NgTemplateOutlet} from '@angular/common';
import {Component, input, output} from '@angular/core';
import {FormGroup, ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faDocker} from '@fortawesome/free-brands-svg-icons';
import {
  faBuildingUser,
  faCheckCircle,
  faCog,
  faDharmachakra,
  faNetworkWired,
  faServer,
  faShip,
} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-deployment-wizard-stepper',
  templateUrl: './deployment-wizard-stepper.component.html',
  providers: [{provide: CdkStepper, useExisting: DeploymentWizardStepperComponent}],
  imports: [CdkStepperModule, ReactiveFormsModule, FaIconComponent, NgTemplateOutlet],
})
export class DeploymentWizardStepperComponent extends CdkStepper {
  public readonly showCustomerStep = input(false);
  public readonly attemptContinueOutput = output<void>({alias: 'attemptContinue'});
  public readonly attemptGoBackOutput = output<void>({alias: 'attemptGoBack'});

  protected readonly faDocker = faDocker;
  protected readonly faDharmachakra = faDharmachakra;
  protected readonly faShip = faShip;
  protected readonly faServer = faServer;
  protected readonly faNetworkWired = faNetworkWired;
  protected readonly faBuildingUser = faBuildingUser;
  protected readonly faCog = faCog;
  protected readonly faCheckCircle = faCheckCircle;

  currentFormGroup() {
    return this.selected!.stepControl as FormGroup;
  }

  // Adjusted for conditional customer step
  getAdjustedIndex(): number {
    return this.showCustomerStep() ? this.selectedIndex : this.selectedIndex + 1;
  }

  isCustomerStep(): boolean {
    return this.showCustomerStep() && this.selectedIndex === 0;
  }

  isApplicationStep(): boolean {
    const adjustedIndex = this.getAdjustedIndex();
    return adjustedIndex === 1;
  }

  isTargetStep(): boolean {
    const adjustedIndex = this.getAdjustedIndex();
    return adjustedIndex === 2;
  }

  isConfigurationStep(): boolean {
    const adjustedIndex = this.getAdjustedIndex();
    return adjustedIndex === 3;
  }

  isConnectStep(): boolean {
    const adjustedIndex = this.getAdjustedIndex();
    return adjustedIndex === 4;
  }

  isFinalStep(): boolean {
    return this.selectedIndex === this.steps.length - 1;
  }

  shouldShowBackButton(): boolean {
    return this.selectedIndex > 0 && !this.isFinalStep();
  }
}
