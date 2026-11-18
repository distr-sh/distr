import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {map} from 'rxjs';
import {refreshInBackground} from '../../../util/refresh';
import {SecretsComponent} from '../../secrets/secrets.component';
import {SecretsService} from '../../services/secrets.service';

@Component({
  selector: 'app-secrets-settings',
  templateUrl: './secrets-settings.component.html',
  imports: [SecretsComponent],
})
export class SecretsSettingsComponent {
  private readonly secretsService = inject(SecretsService);

  protected readonly secrets = toSignal(
    this.secretsService
      .list()
      .pipe(map((secrets) => secrets.filter((secret) => secret.customerOrganizationId === undefined)))
  );

  constructor() {
    refreshInBackground(this.secretsService.refresh());
  }
}
