import {CdkConnectedOverlay, CdkOverlayOrigin} from '@angular/cdk/overlay';
import {LowerCasePipe, TitleCasePipe} from '@angular/common';
import {Component, inject, input, signal} from '@angular/core';
import {RouterLink, RouterLinkActive} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {NavigationEntry} from '../../navigation/navigation';
import {SidebarService} from '../../services/sidebar.service';
import {PlanBadgeComponent} from '../plan-badge.component';
import {PlanFeatureHintComponent} from '../plan-feature-hint.component';

@Component({
  selector: 'app-side-bar-item',
  templateUrl: './side-bar-item.component.html',
  styleUrl: './side-bar-item.component.scss',
  imports: [
    RouterLink,
    RouterLinkActive,
    FaIconComponent,
    CdkOverlayOrigin,
    CdkConnectedOverlay,
    PlanBadgeComponent,
    PlanFeatureHintComponent,
    LowerCasePipe,
    TitleCasePipe,
  ],
})
export class SideBarItemComponent {
  public readonly entry = input.required<NavigationEntry>();

  protected readonly sidebar = inject(SidebarService);

  protected readonly faChevronDown = faChevronDown;
  protected readonly isExpanded = signal(true);
  protected readonly isPlanHintOpen = signal(false);
}
