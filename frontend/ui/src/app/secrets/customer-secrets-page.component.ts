import {OverlayModule} from '@angular/cdk/overlay';
import {ChangeDetectionStrategy, Component, computed, ElementRef, inject, signal, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import {faBoxesStacked, faChevronDown} from '@fortawesome/free-solid-svg-icons';
import {combineLatest, map, startWith, Subject, switchMap} from 'rxjs';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {SecretsService} from '../services/secrets.service';
import {SecretsComponent} from './secrets.component';

@Component({
  templateUrl: './customer-secrets-page.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [SecretsComponent, RouterLink, FontAwesomeModule, OverlayModule],
})
export class CustomerSecretsPageComponent {
  protected readonly faBoxesStacked = faBoxesStacked;
  protected readonly faChevronDown = faChevronDown;

  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly secretsService = inject(SecretsService);
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );
  protected readonly customerOrganizations = toSignal(this.customerOrganizationsService.getCustomerOrganizations());
  protected readonly customerOrganization = computed(() => {
    const id = this.customerOrganizationId();
    return this.customerOrganizations()?.find((org) => org.id === id);
  });

  protected readonly refresh$ = new Subject<void>();

  protected readonly secrets = toSignal(
    combineLatest([
      this.refresh$.pipe(
        startWith(undefined),
        switchMap(() => this.secretsService.list())
      ),
      toObservable(this.customerOrganizationId),
    ]).pipe(
      map(([secrets, customerOrganizationId]) =>
        secrets.filter((it) => it.customerOrganizationId === customerOrganizationId)
      )
    )
  );

  protected readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly breadcrumbDropdown = signal(false);
  breadcrumbDropdownWidth = 0;

  protected toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((v) => !v);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth = this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width;
    }
  }
}
