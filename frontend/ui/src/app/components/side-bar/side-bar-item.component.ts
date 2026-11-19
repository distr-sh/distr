import {CdkOverlayOrigin} from '@angular/cdk/overlay';
import {LowerCasePipe, TitleCasePipe} from '@angular/common';
import {Component, inject, input} from '@angular/core';
import {RouterLink, RouterLinkActive} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {NavigationEntry} from '../../navigation/navigation';
import {SidebarService} from '../../services/sidebar.service';
import {PlanBadgeComponent} from '../plan-badge.component';
import {PlanHintOverlayComponent} from '../plan-hint-overlay.component';

@Component({
  selector: 'app-side-bar-item',
  templateUrl: './side-bar-item.component.html',
  styleUrl: './side-bar-item.component.scss',
  imports: [
    RouterLink,
    RouterLinkActive,
    FaIconComponent,
    CdkOverlayOrigin,
    PlanBadgeComponent,
    PlanHintOverlayComponent,
    LowerCasePipe,
    TitleCasePipe,
  ],
})
export class SideBarItemComponent {
  public readonly entry = input.required<NavigationEntry>();

  protected readonly sidebar = inject(SidebarService);
}
