import {Directive, effect, ElementRef, inject, input, OnDestroy, output} from '@angular/core';

@Directive({selector: '[appIntersect]'})
export class IntersectionObserverDirective implements OnDestroy {
  public readonly intersectEnabled = input(true);
  public readonly threshold = input(0.1);
  public readonly appIntersect = output<boolean>();

  private readonly el = inject(ElementRef<HTMLElement>);
  private observer: IntersectionObserver | null = null;

  constructor() {
    effect(() => {
      this.observer?.disconnect();
      this.observer = null;

      if (this.intersectEnabled()) {
        this.observer = new IntersectionObserver(
          (entries) => {
            for (const entry of entries) {
              this.appIntersect.emit(entry.isIntersecting);
            }
          },
          {threshold: this.threshold()}
        );
        this.observer.observe(this.el.nativeElement);
      }
    });
  }

  ngOnDestroy() {
    this.observer?.disconnect();
  }
}
