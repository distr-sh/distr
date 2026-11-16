import {Component, computed, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {ActivatedRoute} from '@angular/router';
import {EMPTY, startWith, Subject, switchMap} from 'rxjs';
import {SecretsComponent} from '../../../secrets/secrets.component';
import {SecretsService} from '../../../services/secrets.service';

@Component({
  templateUrl: './customer-secrets-tab.component.html',
  imports: [SecretsComponent],
})
export class CustomerSecretsTabComponent {
  private readonly secretsService = inject(SecretsService);

  private readonly parentParams = toSignal(inject(ActivatedRoute).parent?.params ?? EMPTY);
  protected readonly customerOrganizationId = computed(
    () => this.parentParams()?.['customerOrganizationId'] as string | undefined
  );

  protected readonly refresh$ = new Subject<void>();
  private readonly allSecrets = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.secretsService.list())
    )
  );
  protected readonly secrets = computed(() =>
    this.allSecrets()?.filter((it) => it.customerOrganizationId === this.customerOrganizationId())
  );
}
