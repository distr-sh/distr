import {OverlayModule} from '@angular/cdk/overlay';
import {Component, output, signal} from '@angular/core';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faEllipsisVertical, faEye} from '@fortawesome/free-solid-svg-icons';

@Component({
  selector: 'app-overview-menu',
  templateUrl: './overview-menu.component.html',
  imports: [OverlayModule, FaIconComponent],
})
export class OverviewMenuComponent {
  public readonly showInstructions = output<void>();

  protected readonly faEllipsisVertical = faEllipsisVertical;
  protected readonly faEye = faEye;

  protected readonly open = signal(false);
}
