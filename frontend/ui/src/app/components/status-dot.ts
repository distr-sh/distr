import { Component, computed, Directive, input, Signal } from '@angular/core';
import { DeploymentTarget } from '@distr-sh/distr-sdk';
import { isStale } from '../../util/model';

type StatusDotStyle = 'unknown' | 'danger' | 'warning' | 'info' | 'ok' | 'ok-circle'

@Directive({
  host: {
    '[class.rounded-full]': 'true',
    '[class.bg-gray-500]': 'style() === "unknown"',
    '[class.bg-red-400]': 'style() === "danger"',
    '[class.bg-yellow-300]': 'style() === "warning"',
    '[class.bg-blue-400]': 'style() === "info"',
    '[class.bg-lime-600]': 'style() === "ok"',
    '[class.border]': 'style() === "ok-circle"',
    '[class.border-3]': 'style() === "ok-circle"',
    '[class.border-lime-600]': 'style() === "ok-circle"',
  }
})
export abstract class AbstractStatusDotDirective {
  protected abstract readonly style: Signal<StatusDotStyle>;
}

@Directive({ selector: '[appStatusDot]' })
export class StatusDotDirective extends AbstractStatusDotDirective {
  public override style = input.required<StatusDotStyle>({ alias: 'appStatusDot' })
}

@Component({
  selector: 'deployment-target-status-dot',
  template: '<div class="size-full" [appStatusDot]="statusStyle()"></div>',
  imports: [StatusDotDirective],
})
export class DeploymentTargetStatusDotComponent {
  public readonly deploymentTarget = input.required<DeploymentTarget>();

  protected readonly statusStyle = computed(() => {
    const s = this.deploymentTarget().currentStatus;
    if (s === undefined) {
      return 'unknown'
    } else if (isStale(s)) {
      return 'warning'
    } else {
      return 'ok'
    }
  })
}
