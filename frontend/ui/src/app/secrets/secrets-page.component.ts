import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowRight, faAsterisk} from '@fortawesome/free-solid-svg-icons';
import {map, of, startWith, Subject, switchMap} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {CustomerOrganizationsService} from '../services/customer-organizations.service';
import {SecretsService} from '../services/secrets.service';
import {SecretsComponent} from './secrets.component';

@Component({
  templateUrl: './secrets-page.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [SecretsComponent, RouterLink, FaIconComponent],
})
export class SecretsPage {
  protected readonly faArrowRight = faArrowRight;
  protected readonly faAsterisk = faAsterisk;

  private readonly secretsService = inject(SecretsService);
  private readonly customerOrganizationsService = inject(CustomerOrganizationsService);
  private readonly auth = inject(AuthService);
  protected readonly refresh$ = new Subject<void>();
  protected readonly secrets = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.secretsService.list()),
      map((secrets) =>
        this.auth.isVendor() ? secrets.filter((secret) => secret.customerOrganizationId === undefined) : secrets
      )
    )
  );

  protected readonly customerOrganizations = toSignal(
    this.auth.isVendor() ? this.customerOrganizationsService.getCustomerOrganizations() : of([])
  );
}
