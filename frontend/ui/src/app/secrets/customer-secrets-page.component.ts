import {ChangeDetectionStrategy, Component, computed, inject} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {combineLatest, map, startWith, Subject, switchMap} from 'rxjs';
import {CustomerBreadcrumbComponent} from '../components/customer-breadcrumb.component';
import {PageComponent} from '../components/page.component';
import {SecretsService} from '../services/secrets.service';
import {SecretsComponent} from './secrets.component';

@Component({
  templateUrl: './customer-secrets-page.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [SecretsComponent, CustomerBreadcrumbComponent, PageComponent],
})
export class CustomerSecretsPageComponent {
  private readonly secretsService = inject(SecretsService);
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);
  protected readonly customerOrganizationId = computed(
    () => this.routeParams()?.['customerOrganizationId'] as string | undefined
  );

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
}
