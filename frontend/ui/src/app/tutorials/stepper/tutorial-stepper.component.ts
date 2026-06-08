import {CdkStepper, CdkStepperModule} from '@angular/cdk/stepper';
import {NgTemplateOutlet} from '@angular/common';
import {ChangeDetectionStrategy, Component} from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faCircle, faCircleCheck} from '@fortawesome/free-regular-svg-icons';

@Component({
  selector: 'app-tutorial-stepper',
  templateUrl: './tutorial-stepper.component.html',
  providers: [{provide: CdkStepper, useExisting: TutorialStepperComponent}],
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [CdkStepperModule, ReactiveFormsModule, FaIconComponent, NgTemplateOutlet],
})
export class TutorialStepperComponent extends CdkStepper {
  protected readonly faCircle = faCircle;
  protected readonly faCircleCheck = faCircleCheck;

  protected isCurrentStep(i: number): boolean {
    return this.selectedIndex === i;
  }
}
