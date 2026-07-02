import {afterNextRender, Directive, ElementRef, inject} from '@angular/core';

@Directive({selector: '[autofocus]'})
export class AutofocusDirective {
  private readonly elementRef = inject<ElementRef<HTMLElement>>(ElementRef);

  constructor() {
    afterNextRender(() => this.elementRef.nativeElement.focus());
  }
}
