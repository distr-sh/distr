import {computed, Directive, ElementRef, inject, input} from '@angular/core';
import {ActivatedRoute, Router} from '@angular/router';

const interactiveElements = 'a, button, input, select, textarea, label, [role="button"]';

/**
 * Makes a whole table row navigate like a link while keeping its text selectable, which a real link
 * stretched over the row would prevent. Clicks on interactive elements inside the row are left to them.
 */
@Directive({
  selector: '[appRowLink]',
  host: {
    tabindex: '0',
    class:
      'relative cursor-pointer focus:outline-none focus-visible:after:absolute focus-visible:after:inset-0 focus-visible:after:pointer-events-none focus-visible:after:rounded-lg focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-[Highlight] focus-visible:after:[outline-color:-webkit-focus-ring-color]!',
    '(click)': 'onClick($event)',
    '(mousedown)': 'onMouseDown($event)',
    '(auxclick)': 'onAuxClick($event)',
    '(keydown.enter)': 'onEnter($event)',
  },
})
export class RowLinkDirective {
  public readonly appRowLink = input.required<string | readonly string[]>();

  private readonly router = inject(Router);
  private readonly route = inject(ActivatedRoute);
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef);

  private readonly urlTree = computed(() => {
    const link = this.appRowLink();
    return this.router.createUrlTree(typeof link === 'string' ? [link] : [...link], {relativeTo: this.route});
  });

  protected onClick(event: MouseEvent) {
    const newTab = event.ctrlKey || event.metaKey;
    if (this.isFromInteractiveElement(event) || (!newTab && this.hasTextSelection())) {
      return;
    }
    this.open(newTab);
  }

  protected onMouseDown(event: MouseEvent) {
    // Prevents the autoscroll some browsers start on a middle click and the table cell selection
    // Firefox starts on a ctrl+click.
    if ((event.button === 1 || event.ctrlKey || event.metaKey) && !this.isFromInteractiveElement(event)) {
      event.preventDefault();
    }
  }

  protected onAuxClick(event: MouseEvent) {
    if (event.button === 1 && !this.isFromInteractiveElement(event)) {
      this.open(true);
    }
  }

  protected onEnter(event: Event) {
    const keyboardEvent = event as KeyboardEvent;
    if (keyboardEvent.target === this.host.nativeElement) {
      this.open(keyboardEvent.ctrlKey || keyboardEvent.metaKey);
    }
  }

  private open(newTab: boolean) {
    if (newTab) {
      window.open(this.router.serializeUrl(this.urlTree()), '_blank');
    } else {
      this.router.navigateByUrl(this.urlTree());
    }
  }

  private isFromInteractiveElement(event: Event): boolean {
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
