import {Component, inject} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {startWith, Subject, switchMap} from 'rxjs';
import {PageComponent} from '../components/page.component';
import {SecretsService} from '../services/secrets.service';
import {SecretsComponent} from './secrets.component';

@Component({
  templateUrl: './secrets-page.component.html',
  imports: [SecretsComponent, PageComponent],
})
export class SecretsPage {
  private readonly secretsService = inject(SecretsService);
  protected readonly refresh$ = new Subject<void>();
  protected readonly secrets = toSignal(
    this.refresh$.pipe(
      startWith(undefined),
      switchMap(() => this.secretsService.list())
    )
  );
}
