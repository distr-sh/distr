import {Directive, effect, ElementRef, inject, input, OnDestroy, output} from '@angular/core';

@Directive({selector: '[appIntersectionObserver]'})
export class IntersectionObserverDirective implements OnDestroy {
  public readonly enabled = input(true);
  public readonly threshold = input(0.1);
  public readonly intersecting = output<boolean>();

  private readonly el = inject(ElementRef<HTMLElement>);
  private observer: IntersectionObserver | null = null;

  constructor() {
    effect(() => {
      this.observer?.disconnect();
      this.observer = null;

      if (this.enabled()) {
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
    });
  }

  ngOnDestroy() {
    this.observer?.disconnect();
  }
}
