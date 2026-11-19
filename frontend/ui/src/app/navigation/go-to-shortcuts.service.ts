import {computed, inject, Injectable} from '@angular/core';
import {Router} from '@angular/router';
import {isEditable} from '../../util/keyboard';
import {NavigationService} from './navigation.service';

const SEQUENCE_TIMEOUT_MS = 1500;

/** Opens the page of a navigation entry when "g" is followed by the entry's shortcut key. */
@Injectable()
export class GoToShortcutsService {
  private readonly navigation = inject(NavigationService);
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
    if (
      event.defaultPrevented ||
      event.repeat ||
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
      if (route) {
        event.preventDefault();
        this.router.navigateByUrl(route);
      }
    } else {
      this.armedAt = key === 'g' ? Date.now() : null;
    }
  }
}
