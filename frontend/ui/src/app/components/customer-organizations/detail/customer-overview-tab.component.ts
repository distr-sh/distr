import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY} from 'rxjs';
import {CustomerOrganizationsService} from '../../../services/customer-organizations.service';
import {CardHeadingComponent} from '../../card-heading.component';
import {CustomerFeaturesComponent} from './customer-features.component';

@Component({
  templateUrl: './customer-overview-tab.component.html',
  imports: [CardHeadingComponent, CustomerFeaturesComponent],
})
export class CustomerOverviewTabComponent {
  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  private readonly customerOrganizations = toSignal(inject(CustomerOrganizationsService).getCustomerOrganizations());
  protected readonly customerOrganization = computed(() =>
    this.customerOrganizations()?.find((it) => it.id === this.parentParams()?.['customerOrganizationId'])
  );
}
