import {ChangeDetectionStrategy, Component, computed, inject, TemplateRef, viewChild} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterOutlet} from '@angular/router';
import {map} from 'rxjs';
import {ContextSwitcherService} from '../navigation/context-switcher.service';
import {GoToShortcutsService} from '../navigation/go-to-shortcuts.service';
import {NavigationService} from '../navigation/navigation.service';
import {AuthService} from '../services/auth.service';
import {OrganizationService} from '../services/organization.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {CommandPaletteModalComponent} from './command-palette/command-palette-modal.component';
import {NavBarComponent} from './nav-bar/nav-bar.component';
import {SideBarComponent} from './side-bar/side-bar.component';

@Component({
  selector: 'app-nav-shell',
  template: `
    <app-nav-bar
      [isSubscriptionBannerVisible]="isSubscriptionBannerVisible()"
      [isSidebarVisible]="isSidebarVisible()" />
    <app-side-bar [isBannerVisible]="isBannerVisible()" [isSidebarVisible]="isSidebarVisible()" />
    <div [class.sm:ml-64]="isSidebarVisible()" [class.pt-16]="!isBannerVisible()" [class.pt-33]="isBannerVisible()">
      <router-outlet />
    </div>
    <ng-template #commandPalette>
      <app-command-palette-modal
        animate.enter="animate-flip-in-top"
        animate.leave="animate-flip-out-top"
        (closed)="commandPaletteRef?.close()" />
    </ng-template>
  `,
  changeDetection: ChangeDetectionStrategy.Eager,
  host: {
    '(document:keydown)': 'goToShortcuts.onKeydown($event)',
    '(document:keydown.control.k)': 'toggleCommandPalette($event)',
    '(document:keydown.meta.k)': 'toggleCommandPalette($event)',
  },
  imports: [NavBarComponent, SideBarComponent, RouterOutlet, CommandPaletteModalComponent],
  providers: [NavigationService, GoToShortcutsService, ContextSwitcherService],
})
export class NavShellComponent {
  private readonly organizationService = inject(OrganizationService);
  private readonly auth = inject(AuthService);
  private readonly overlay = inject(OverlayService);
  protected readonly goToShortcuts = inject(GoToShortcutsService);
  private readonly switcher = inject(ContextSwitcherService);

  private readonly commandPalette = viewChild.required<TemplateRef<unknown>>('commandPalette');
  protected commandPaletteRef?: DialogRef;

  protected readonly organization$ = this.organizationService.get();

  protected readonly isSubscriptionExpired = this.organizationService.isSubscriptionExpired;

  private readonly isOrganizationLoaded = toSignal(this.organization$.pipe(map(() => true)), {initialValue: false});

  private readonly isSubscriptionTrial = toSignal(
    this.organization$.pipe(map((org) => org.subscriptionType === 'trial')),
    {initialValue: false}
  );

  protected readonly isSidebarVisible = computed(() => !this.isSubscriptionExpired());

  protected readonly isSubscriptionBannerVisible = computed(() => {
    if (!this.auth.isVendor()) {
      return false;
    }
    return this.isSubscriptionExpired() || this.isSubscriptionTrial();
  });

  // A session in a lens is never a vendor session, so the two banners never show at once.
  protected readonly isBannerVisible = computed(
    () => (this.auth.isLens() && this.isOrganizationLoaded()) || this.isSubscriptionBannerVisible()
  );

  protected toggleCommandPalette(event: Event) {
    // Ctrl+K and Cmd+K are the browsers' own shortcuts for their search and address bars
    event.preventDefault();
    if (event instanceof KeyboardEvent && event.repeat) {
      return;
    }
    if (this.commandPaletteRef) {
      this.commandPaletteRef.close();
      return;
    }
    // A command that navigates would leave the modal open on the next page.
    if (document.querySelector('.cdk-overlay-backdrop')) {
      return;
    }
    this.switcher.refreshLenses();
    const dialogRef = this.overlay.showModal(this.commandPalette());
    this.commandPaletteRef = dialogRef;
    dialogRef.closed().subscribe(() => {
      if (this.commandPaletteRef === dialogRef) {
        this.commandPaletteRef = undefined;
      }
    });
  }
}
