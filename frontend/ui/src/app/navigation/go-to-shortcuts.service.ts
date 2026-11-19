import {computed, inject, Injectable} from '@angular/core';
import {Router} from '@angular/router';
import {isEditable} from '../../util/keyboard';
import {ContextSwitcherService} from './context-switcher.service';
import {NavigationService} from './navigation.service';

const SEQUENCE_TIMEOUT_MS = 1500;

/** Leaves the customer lens when pressed after "g". No portal's navigation may use it. */
export const LEAVE_LENS_SHORTCUT = 'p';

/** Opens the page of a navigation entry when "g" is followed by the entry's shortcut key. */
@Injectable()
export class GoToShortcutsService {
  private readonly navigation = inject(NavigationService);
  private readonly switcher = inject(ContextSwitcherService);
  private readonly router = inject(Router);

  private readonly routesByShortcut = computed(
    () =>
      new Map(
        this.navigation
          .sections()
          .flatMap((section) => section.entries)
          .flatMap((entry): [string, string][] =>
            entry.kind === 'link' && entry.shortcut ? [[entry.shortcut, entry.route]] : []
          )
      )
  );

  private armedAt: number | null = null;

  public onKeydown(event: KeyboardEvent): void {
    if (event.repeat) {
      return;
    }
    if (
      event.defaultPrevented ||
      event.ctrlKey ||
      event.metaKey ||
      event.altKey ||
      isEditable(event.target) ||
      document.querySelector('.cdk-overlay-backdrop')
    ) {
      this.armedAt = null;
      return;
    }
    const key = event.key.toLowerCase();
    if (this.armedAt !== null && Date.now() - this.armedAt <= SEQUENCE_TIMEOUT_MS) {
      this.armedAt = null;
      const route = this.routesByShortcut().get(key);
      if (key === LEAVE_LENS_SHORTCUT && this.switcher.lensHome()) {
        event.preventDefault();
        this.switcher.leaveLens();
      } else if (route) {
        event.preventDefault();
        this.router.navigateByUrl(route);
      }
    } else {
      this.armedAt = key === 'g' ? Date.now() : null;
    }
  }
}
