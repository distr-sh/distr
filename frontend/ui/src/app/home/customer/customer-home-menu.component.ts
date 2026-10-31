import {OverlayModule} from '@angular/cdk/overlay';
import {Component, input, output, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faEllipsisVertical, faEye, faListCheck} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-customer-home-menu',
  templateUrl: './customer-home-menu.component.html',
  imports: [OverlayModule, FaIconComponent],
})
export class CustomerHomeMenuComponent {
  public readonly instructionsHidden = input.required<boolean>();
  public readonly checklistHidden = input.required<boolean>();
  public readonly showInstructions = output<void>();
  public readonly showChecklist = output<void>();

  protected readonly faEllipsisVertical = faEllipsisVertical;
  protected readonly faEye = faEye;
  protected readonly faListCheck = faListCheck;

  protected readonly open = signal(false);
}
