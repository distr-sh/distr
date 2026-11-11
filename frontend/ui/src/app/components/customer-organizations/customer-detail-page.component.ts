import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {CustomerBreadcrumbComponent} from '../customer-breadcrumb.component';
import {PageComponent} from '../page.component';

@Component({
  templateUrl: './customer-detail-page.component.html',
  imports: [CustomerBreadcrumbComponent, PageComponent],
})
export class CustomerDetailPageComponent {
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );
}
