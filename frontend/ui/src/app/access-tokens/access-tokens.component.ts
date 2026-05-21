import {Component, inject} from '@angular/core';
import {AccessTokensService} from '../services/access-tokens.service';
import {AccessTokensTableComponent} from './access-tokens-table.component';

@Component({
  selector: 'app-access-tokens',
  imports: [AccessTokensTableComponent],
  template: `<section class="bg-gray-50 dark:bg-gray-900 p-3 sm:p-5 antialiased">
    <div class="mx-auto max-w-screen-2xl px-4 lg:px-12">
      <app-access-tokens-table
        [store]="accessTokens"
        drawerTitle="Create a Personal Access Token"
        keyDisplayText="Your Personal Access Token:" />
    </div>
  </section>`,
})
export class AccessTokensComponent {
  protected readonly accessTokens = inject(AccessTokensService);
}
