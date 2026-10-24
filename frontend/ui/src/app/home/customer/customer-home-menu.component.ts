import {OverlayModule} from '@angular/cdk/overlay';
import {Component, output, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faEllipsisVertical, faEye} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-customer-home-menu',
  templateUrl: './customer-home-menu.component.html',
  imports: [OverlayModule, FaIconComponent],
})
export class CustomerHomeMenuComponent {
  public readonly showInstructions = output<void>();

  protected readonly faEllipsisVertical = faEllipsisVertical;
  protected readonly faEye = faEye;

  protected readonly open = signal(false);
}
