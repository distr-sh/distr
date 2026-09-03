import {afterNextRender, booleanAttribute, Directive, ElementRef, inject, input} from '@angular/core';

@Directive({selector: '[autofocus]'})
export class AutofocusDirective {
  private readonly elementRef = inject<ElementRef<HTMLElement>>(ElementRef);

  // a bare `autofocus` attribute passes the empty string, which the transform reads as true
  public readonly autofocus = input(true, {transform: booleanAttribute});

  constructor() {
    afterNextRender(() => {
      if (this.autofocus()) {
        this.elementRef.nativeElement.focus();
      }
    });
  }
}
