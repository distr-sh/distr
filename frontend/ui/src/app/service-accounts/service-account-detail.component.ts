import {DatePipe, TitleCasePipe} from '@angular/common';
import {Component, computed, inject, signal} from '@angular/core';
import {toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule, Validators} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {AccountRole} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faArrowLeft} from '@fortawesome/free-solid-svg-icons';
import {firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {AccessTokensTableComponent, AccessTokenStore} from '../access-tokens/access-tokens-table.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {ServiceAccountsService} from '../services/service-accounts.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-service-account-detail',
  imports: [
    AccessTokensTableComponent,
    AutotrimDirective,
    DatePipe,
    FaIconComponent,
    ReactiveFormsModule,
    RouterLink,
    TitleCasePipe,
  ],
  templateUrl: './service-account-detail.component.html',
})
export class ServiceAccountDetailComponent {
  protected readonly faArrowLeft = faArrowLeft;

  private readonly route = inject(ActivatedRoute);
  private readonly service = inject(ServiceAccountsService);
  private readonly toast = inject(ToastService);

  protected readonly serviceAccountId = computed(() => this.route.snapshot.paramMap.get('serviceAccountId') ?? '');

  private readonly refreshSA$ = new Subject<void>();
  protected readonly serviceAccount = toSignal(
    this.refreshSA$.pipe(
      startWith(0),
      switchMap(() => this.service.get(this.serviceAccountId()))
    )
  );

  protected readonly tokenStore: AccessTokenStore = {
    list: () => this.service.listTokens(this.serviceAccountId()),
    create: (request) => this.service.createToken(this.serviceAccountId(), request),
    delete: (tokenId) => this.service.deleteToken(this.serviceAccountId(), tokenId),
  };

  protected readonly editForm = new FormGroup({
    name: new FormControl<string>('', {nonNullable: true, validators: [Validators.required]}),
    accountRole: new FormControl<AccountRole>('read_write', {nonNullable: true}),
  });
  protected editLoading = signal(false);
  protected editing = signal(false);

  public startEdit() {
    const sa = this.serviceAccount();
    if (!sa) {
      return;
    }
    this.editForm.reset({name: sa.name, accountRole: sa.accountRole});
    this.editing.set(true);
  }

  public cancelEdit() {
    this.editing.set(false);
  }

  public async saveEdit() {
    const sa = this.serviceAccount();
    if (!sa || this.editForm.invalid) {
      return;
    }
    this.editLoading.set(true);
    try {
      await firstValueFrom(
        this.service.patch(sa.id, {
          name: this.editForm.value.name,
          accountRole: this.editForm.value.accountRole,
        })
      );
      this.toast.success('Service account updated');
      this.editing.set(false);
      this.refreshSA$.next();
    } finally {
      this.editLoading.set(false);
    }
  }
}
