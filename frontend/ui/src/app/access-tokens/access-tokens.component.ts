import {OverlayModule} from '@angular/cdk/overlay';
import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, computed, inject, TemplateRef} from '@angular/core';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {AccessToken, AccessTokenWithKey, CreateAccessTokenRequest, UserRole} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard, faMagnifyingGlass, faPlus, faTrash, faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {firstValueFrom, startWith, Subject, switchMap} from 'rxjs';
import {isExpired, RelativeDatePipe} from '../../util/dates';
import {UserRoleLabelPipe} from '../../util/user-role';
import {CreatedAccessTokenComponent} from '../components/created-access-token.component';
import {UserRoleSelectComponent} from '../components/user-role-select.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AccessTokensService} from '../services/access-tokens.service';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-access-tokens',
  imports: [
    ReactiveFormsModule,
    FaIconComponent,
    AsyncPipe,
    DatePipe,
    AutotrimDirective,
    OverlayModule,
    RelativeDatePipe,
    CreatedAccessTokenComponent,
    UserRoleSelectComponent,
    UserRoleLabelPipe,
  ],
  templateUrl: './access-tokens.component.html',
})
export class AccessTokensComponent {
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faTrash = faTrash;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly faClipboard = faClipboard;

  private readonly accessTokens = inject(AccessTokensService);
  private readonly auth = inject(AuthService);
  private readonly refresh$ = new Subject<void>();
  protected readonly accessTokens$ = this.refresh$.pipe(
    startWith(0),
    switchMap(() => this.accessTokens.list())
  );

  private readonly toast = inject(ToastService);

  private readonly overlay = inject(OverlayService);
  protected drawer: DialogRef<void> | null = null;

  protected readonly currentUserRole = computed<UserRole | undefined>(() => this.auth.getClaims()?.role);

  protected readonly editForm = new FormGroup({
    label: new FormControl('', {nonNullable: true}),
    expiresAt: new FormControl('', {nonNullable: true}),
    userRole: new FormControl<UserRole | undefined>(undefined),
  });

  protected editFormLoading = false;
  protected createdToken: AccessTokenWithKey | null = null;

  public openDrawer(template: TemplateRef<unknown>) {
    this.hideDrawer();
    this.editForm.patchValue({
      label: '',
      expiresAt: dayjs()
        .add(dayjs.duration({days: 30}))
        .format('YYYY-MM-DD'),
      userRole: undefined,
    });
    this.drawer = this.overlay.showDrawer(template);
  }

  public hideDrawer() {
    this.drawer?.dismiss();
  }

  public async createAccessToken() {
    this.editFormLoading = true;
    const request: CreateAccessTokenRequest = {};
    if (this.editForm.value.label) {
      request.label = this.editForm.value.label;
    }
    if (this.editForm.value.expiresAt) {
      request.expiresAt = new Date(this.editForm.value.expiresAt);
    }
    if (this.editForm.value.userRole) {
      request.userRole = this.editForm.value.userRole;
    }
    try {
      this.createdToken = await firstValueFrom(this.accessTokens.create(request));
      this.toast.success('token created');
      this.hideDrawer();
      this.refresh$.next();
    } finally {
      this.editFormLoading = false;
    }
  }

  public async deleteAccessToken(accessToken: AccessToken) {
    if (await firstValueFrom(this.overlay.confirm(`Really delete token '${accessToken.label}'?`))) {
      try {
        await firstValueFrom(this.accessTokens.delete(accessToken.id!));
        this.refresh$.next();
      } catch (e) {}
    }
  }

  protected readonly isExpired = isExpired;
}
