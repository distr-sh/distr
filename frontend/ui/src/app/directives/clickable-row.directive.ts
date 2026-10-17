import {computed, Directive, ElementRef, inject, input, output} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';

const interactiveElements = 'a, button, input, select, textarea, label, [role="button"]';

/**
 * Makes a whole table row open its details while keeping its text selectable, which a real link
 * stretched over the row would prevent. With a link the row navigates, otherwise it emits `rowActivate`
 * for the caller to open a drawer or modal. Clicks on interactive elements inside the row are left to them.
 */
@Directive({
  selector: '[appClickableRow]',
  host: {
    '[attr.tabindex]': 'appClickableRowDisabled() ? null : 0',
    '[class.cursor-pointer]': '!appClickableRowDisabled()',
    class:
      'relative focus:outline-none focus-visible:after:absolute focus-visible:after:inset-0 focus-visible:after:pointer-events-none focus-visible:after:rounded-lg focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-[Highlight] focus-visible:after:[outline-color:-webkit-focus-ring-color]!',
    '(click)': 'onClick($event)',
    '(mousedown)': 'onMouseDown($event)',
    '(auxclick)': 'onAuxClick($event)',
    '(keydown.enter)': 'onEnter($event)',
  },
})
export class ClickableRowDirective {
  public readonly appClickableRow = input<string | readonly string[]>('');
  public readonly appClickableRowDisabled = input(false);
  public readonly rowActivate = output<void>();

  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  private readonly urlTree = computed(() => {
    const link = this.appClickableRow();
    if (link.length === 0) {
      return undefined;
    }
    return this.router.createUrlTree(typeof link === 'string' ? [link] : [...link], {relativeTo: this.route});
  });

  protected onClick(event: MouseEvent) {
    const newTab = event.ctrlKey || event.metaKey;
    if (this.isIgnored(event) || (!newTab && this.hasTextSelection())) {
      return;
    }
    this.activate(newTab);
  }

  protected onMouseDown(event: MouseEvent) {
    // Prevents the autoscroll some browsers start on a middle click and the table cell selection
    // Firefox starts on a ctrl+click.
    if ((event.button === 1 || event.ctrlKey || event.metaKey) && !this.isIgnored(event)) {
      event.preventDefault();
    }
  }

  protected onAuxClick(event: MouseEvent) {
    if (event.button === 1 && this.urlTree() && !this.isIgnored(event)) {
      this.activate(true);
    }
  }

  protected onEnter(event: Event) {
    const keyboardEvent = event as KeyboardEvent;
    if (keyboardEvent.target === this.host.nativeElement && !this.appClickableRowDisabled()) {
      this.activate(keyboardEvent.ctrlKey || keyboardEvent.metaKey);
    }
  }

  private activate(newTab: boolean) {
    const urlTree = this.urlTree();
    if (!urlTree) {
      this.rowActivate.emit();
    } else if (newTab) {
      window.open(this.router.serializeUrl(urlTree), '_blank');
    } else {
      this.router.navigateByUrl(urlTree);
    }
  }

  private isIgnored(event: Event): boolean {
    if (this.appClickableRowDisabled()) {
      return true;
    }
    const interactive = (event.target as Element).closest(interactiveElements);
    return (
      interactive !== null && interactive !== this.host.nativeElement && this.host.nativeElement.contains(interactive)
    );
  }

  private hasTextSelection(): boolean {
    const selection = window.getSelection();
    return selection !== null && !selection.isCollapsed && this.host.nativeElement.contains(selection.anchorNode);
  }
}
