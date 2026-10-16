import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {faAsterisk} from '@fortawesome/free-solid-svg-icons';
import {map, startWith, Subject, switchMap} from 'rxjs';
import {OrganizationScopeNavComponent} from '../components/organization-scope-nav.component';
import {PageComponent} from '../components/page.component';
import {AuthService} from '../services/auth.service';
import {SecretsService} from '../services/secrets.service';
import {SecretsComponent} from './secrets.component';

@Component({
  templateUrl: './secrets-page.component.html',
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [SecretsComponent, OrganizationScopeNavComponent, PageComponent],
})
export class SecretsPage {
  protected readonly faAsterisk = faAsterisk;

  private readonly secretsService = inject(SecretsService);
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
}
