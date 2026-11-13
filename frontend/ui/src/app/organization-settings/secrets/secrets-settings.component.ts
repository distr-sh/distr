import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map, startWith, Subject, switchMap} from 'rxjs';
import {SecretsComponent} from '../../secrets/secrets.component';
import {SecretsService} from '../../services/secrets.service';

@Component({
  selector: 'app-secrets-settings',
  templateUrl: './secrets-settings.component.html',
  imports: [SecretsComponent],
})
export class SecretsSettingsComponent {
  private readonly secretsService = inject(SecretsService);
  protected readonly refresh$ = new Subject<void>();

  protected readonly secrets = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.secretsService.list()),
      map((secrets) => secrets.filter((secret) => secret.customerOrganizationId === undefined))
    ),
    {initialValue: []}
  );
}
