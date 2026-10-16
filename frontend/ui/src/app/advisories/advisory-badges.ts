import {Component, computed, input} from '@angular/core';
import {AdvisoryImpactState, AdvisorySeverity, AdvisoryStatus} from '@distr-sh/distr-sdk';
import {
  affectedLabel,
  affectedVariant,
  impactStateLabel,
  impactStateVariant,
  severityLabel,
  severityVariant,
  statusLabel,
  statusVariant,
} from './advisory-display';

@Component({
  selector: 'app-advisory-severity-badge',
  host: {class: 'distr-badge', '[class]': 'variantClass()'},
  template: `{{ label() }}`,
})
export class AdvisorySeverityBadgeComponent {
  public readonly severity = input.required<AdvisorySeverity>();
  protected readonly label = computed(() => severityLabel(this.severity()));
  protected readonly variantClass = computed(() => `distr-badge-${severityVariant(this.severity())}`);
}

@Component({
  selector: 'app-advisory-status-badge',
  host: {class: 'distr-badge', '[class]': 'variantClass()'},
  template: `{{ label() }}`,
})
export class AdvisoryStatusBadgeComponent {
  public readonly status = input.required<AdvisoryStatus>();
  protected readonly label = computed(() => statusLabel(this.status()));
  protected readonly variantClass = computed(() => `distr-badge-${statusVariant(this.status())}`);
}

@Component({
  selector: 'app-advisory-affected-badge',
  host: {class: 'distr-badge', '[class]': 'variantClass()'},
  template: `{{ label() }}`,
})
export class AdvisoryAffectedBadgeComponent {
  public readonly affected = input.required<boolean | undefined>();
  protected readonly label = computed(() => affectedLabel(this.affected()));
  protected readonly variantClass = computed(() => `distr-badge-${affectedVariant(this.affected())}`);
}

@Component({
  selector: 'app-advisory-impact-state-badge',
  host: {class: 'distr-badge', '[class]': 'variantClass()'},
  template: `{{ label() }}`,
})
export class AdvisoryImpactStateBadgeComponent {
  public readonly state = input.required<AdvisoryImpactState>();
  protected readonly label = computed(() => impactStateLabel(this.state()));
  protected readonly variantClass = computed(() => `distr-badge-${impactStateVariant(this.state())}`);
}
