import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, inject, input, signal, TemplateRef} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {AccessToken, AccessTokenWithKey, CreateAccessTokenRequest} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {firstValueFrom, Observable, startWith, Subject, switchMap} from 'rxjs';
import {isExpired, RelativeDatePipe} from '../../util/dates';
import {ClipComponent} from '../components/clip.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';

export interface AccessTokenStore {
  list(): Observable<AccessToken[]>;
  create(request: CreateAccessTokenRequest): Observable<AccessTokenWithKey>;
  delete(id: string): Observable<void>;
}

@Component({
  selector: 'app-access-tokens-table',
  imports: [
    AsyncPipe,
    AutotrimDirective,
    ClipComponent,
    DatePipe,
    FaIconComponent,
    OverlayModule,
    ReactiveFormsModule,
    RelativeDatePipe,
  ],
  templateUrl: './access-tokens-table.component.html',
})
export class AccessTokensTableComponent {
  public readonly store = input.required<AccessTokenStore>();
  public readonly drawerTitle = input<string>('Create access token');
  public readonly keyDisplayText = input<string>('Your access token:');

  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faXmark = faXmark;
  protected readonly isExpired = isExpired;

  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);

  private readonly refresh$ = new Subject<void>();
  protected readonly accessTokens$ = this.refresh$.pipe(
    startWith(0),
    switchMap(() => this.store().list())
  );

  protected readonly editForm = new FormGroup({
    label: new FormControl('', {nonNullable: true}),
    expiresAt: new FormControl('', {nonNullable: true}),
  });
  protected editFormLoading = signal(false);
  protected createdToken = signal<AccessTokenWithKey | null>(null);
  protected drawer: DialogRef<void> | null = null;

  public openDrawer(template: TemplateRef<unknown>) {
    this.hideDrawer();
    this.editForm.patchValue({
      label: '',
      expiresAt: dayjs()
        .add(dayjs.duration({days: 30}))
        .format('YYYY-MM-DD'),
    });
    this.drawer = this.overlay.showDrawer(template);
  }

  public hideDrawer() {
    this.drawer?.dismiss();
  }

  public async createAccessToken() {
    this.editFormLoading.set(true);
    const request: CreateAccessTokenRequest = {};
    if (this.editForm.value.label) {
      request.label = this.editForm.value.label;
    }
    if (this.editForm.value.expiresAt) {
      request.expiresAt = new Date(this.editForm.value.expiresAt);
    }
    try {
      this.createdToken.set(await firstValueFrom(this.store().create(request)));
      this.toast.success('Token created');
      this.hideDrawer();
      this.refresh$.next();
    } finally {
      this.editFormLoading.set(false);
    }
  }

  public async deleteAccessToken(accessToken: AccessToken) {
    if (await firstValueFrom(this.overlay.confirm(`Really delete token '${accessToken.label}'?`))) {
      try {
        await firstValueFrom(this.store().delete(accessToken.id!));
        this.refresh$.next();
      } catch {
        // toast handled globally
      }
    }
  }
}
