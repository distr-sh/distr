import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {Subject, map, startWith, switchMap} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {SecretsService} from '../services/secrets.service';
import {SecretsComponent} from './secrets.component';

@Component({
  template: `<section class="bg-gray-50 dark:bg-gray-900 p-3 sm:p-5 antialiased">
    <div class="mx-auto max-w-screen-2xl px-4 lg:px-12">
      <div class="bg-white dark:bg-gray-800 relative shadow-md sm:rounded-lg overflow-hidden">
        <app-secrets [secrets]="secrets() ?? []" (refresh)="refresh$.next()" />
      </div>
    </div>
  </section>`,
  changeDetection: ChangeDetectionStrategy.Eager,
  imports: [SecretsComponent],
})
export class SecretsPage {
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
