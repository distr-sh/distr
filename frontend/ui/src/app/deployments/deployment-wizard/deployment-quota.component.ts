import {Component, computed, input} from '@angular/core';
import {UNLIMITED_QTY} from '../../types/subscription';

@Component({
  selector: 'app-deployment-quota',
  templateUrl: './deployment-quota.component.html',
})
export class DeploymentQuotaComponent {
  public readonly usage = input.required<number>();
  public readonly limit = input.required<number>();

  protected readonly isLimited = computed(() => this.limit() !== UNLIMITED_QTY);
}
