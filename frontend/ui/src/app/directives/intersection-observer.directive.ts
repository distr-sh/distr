import {Directive, ElementRef, inject, input, OnDestroy, output} from '@angular/core';

@Directive({selector: '[appIntersectionObserver]'})
export class IntersectionObserverDirective implements OnDestroy {
  public readonly threshold = input(0.1);
  public readonly intersecting = output<boolean>();

  private readonly el = inject(ElementRef<HTMLElement>);
  private readonly observer: IntersectionObserver;

  constructor() {
    this.observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          this.intersecting.emit(entry.isIntersecting);
        }
      },
      {threshold: this.threshold()}
    );
    this.observer.observe(this.el.nativeElement);
  }

  ngOnDestroy() {
    this.observer.disconnect();
  }
}
