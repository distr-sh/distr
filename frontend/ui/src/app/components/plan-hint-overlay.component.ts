import {CdkConnectedOverlay, CdkOverlayOrigin, ConnectedPosition} from '@angular/cdk/overlay';
import {Component, computed, effect, inject, input, Renderer2, signal} from '@angular/core';
import {UpgradePlan} from '../types/subscription';
import {PlanFeatureHintComponent} from './plan-feature-hint.component';

export type PlanHintPlacement = 'end' | 'below';

/** The plan hint of a locked element, toggled by a click on `origin` and closed by a click anywhere else. */
@Component({
  selector: 'app-plan-hint-overlay',
  templateUrl: './plan-hint-overlay.component.html',
  imports: [CdkConnectedOverlay, PlanFeatureHintComponent],
})
export class PlanHintOverlayComponent {
  private readonly renderer = inject(Renderer2);

  public readonly origin = input.required<CdkOverlayOrigin>();
  public readonly plan = input.required<UpgradePlan>();
  public readonly placement = input<PlanHintPlacement>('below');

  protected readonly open = signal(false);

  protected readonly positions = computed<ConnectedPosition[]>(() => {
    switch (this.placement()) {
      case 'end':
        return [{originX: 'end', overlayX: 'start', originY: 'center', overlayY: 'center'}];
      case 'below':
        return [{originX: 'center', overlayX: 'center', originY: 'bottom', overlayY: 'top'}];
    }
  });

  constructor() {
    // The origin belongs to the caller's template, so the click that opens the hint is not one this template sees.
    effect((onCleanup) => {
      onCleanup(
        this.renderer.listen(this.origin().elementRef.nativeElement, 'click', () => this.open.update((open) => !open))
      );
    });
  }
}
