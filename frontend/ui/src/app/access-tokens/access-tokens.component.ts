import {OverlayModule} from '@angular/cdk/overlay';
import {DatePipe} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef} from '@angular/core';
import {rxResource, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {Router, RouterLink} from '@angular/router';
import {AccessToken, AccessTokenSecret, CreateAccessTokenRequest, UserRole} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faClipboard, faPen, faPlus, faTrash, faTriangleExclamation, faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {firstValueFrom} from 'rxjs';
import {isExpired, RelativeDatePipe} from '../../util/dates';
import {getFormDisplayedError} from '../../util/errors';
import {USER_ROLE_LABELS, UserRoleLabelPipe} from '../../util/user-role';
import {ExpiresAtPickerComponent} from '../components/expires-at-picker/expires-at-picker.component';
import {PageComponent} from '../components/page.component';
import {SearchBarComponent} from '../components/search-bar.component';
import {UserRoleSelectComponent} from '../components/user-role-select.component';
import {AutotrimDirective} from '../directives/autotrim.directive';
import {AccessTokensService} from '../services/access-tokens.service';
import {AuthService} from '../services/auth.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {accessTokenName} from './access-token-name';

interface AccessTokenRow {
  token: AccessToken;
  name: string;
  // Everything the filter matches against: the label, and the key and id so that one pasted from
  // elsewhere finds its token.
  search: string;
  expired: boolean;
  // A token without secrets predates them and is stored in plain text. Its first secret can only
  // be added at the cost of invalidating the token that is in circulation.
  legacy: boolean;
  secrets: AccessTokenSecret[];
}

@Component({
  selector: 'app-access-tokens',
  imports: [
    ReactiveFormsModule,
    FaIconComponent,
    DatePipe,
    AutotrimDirective,
    OverlayModule,
    RouterLink,
    RelativeDatePipe,
    ExpiresAtPickerComponent,
    SearchBarComponent,
    UserRoleSelectComponent,
    UserRoleLabelPipe,
    PageComponent,
  ],
  templateUrl: './access-tokens.component.html',
})
export class AccessTokensComponent {
  protected readonly faPen = faPen;
  protected readonly faTrash = faTrash;
  protected readonly faPlus = faPlus;
  protected readonly faXmark = faXmark;
  protected readonly faClipboard = faClipboard;
  protected readonly faTriangleExclamation = faTriangleExclamation;

  private readonly accessTokensService = inject(AccessTokensService);
  private readonly auth = inject(AuthService);
  private readonly toast = inject(ToastService);
  private readonly overlay = inject(OverlayService);
  private readonly router = inject(Router);

  private readonly accessTokens = rxResource({stream: () => this.accessTokensService.list()});

  private readonly rows = computed<AccessTokenRow[]>(() =>
    (this.accessTokens.value() ?? []).map((token) => ({
      token,
      name: accessTokenName(token),
      search: `${token.label ?? ''} ${token.keyId} ${token.id}`.toLowerCase(),
      expired: isExpired(token),
      legacy: token.secrets.length === 0,
      secrets: token.secrets,
    }))
  );

  protected readonly filterForm = new FormGroup({search: new FormControl('', {nonNullable: true})});
  private readonly filterValue = toSignal(this.filterForm.controls.search.valueChanges);

  protected readonly filteredRows = computed(() => {
    const value = this.filterValue()?.toLowerCase();
    const rows = this.rows();
    return !value ? rows : rows.filter((row) => row.search.includes(value));
  });

  protected drawer: DialogRef<void> | null = null;

  protected readonly currentUserRole = computed<UserRole | undefined>(() => this.auth.getClaims()?.role);
  protected readonly inheritOptionLabel = computed(() => {
    const role = this.currentUserRole();
    return role ? `Inherit (${USER_ROLE_LABELS[role]})` : 'Inherit from my role';
  });

  protected readonly editForm = new FormGroup({
    label: new FormControl('', {nonNullable: true}),
    expiresAt: new FormControl('', {nonNullable: true}),
    userRole: new FormControl<UserRole | undefined>(undefined),
  });

  protected readonly editFormLoading = signal(false);

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
    this.editFormLoading.set(true);
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
      const created = await firstValueFrom(this.accessTokensService.create(request));
      this.toast.success('token created');
      this.hideDrawer();
      // The new token travels in the navigation because the server never returns it again.
      await this.router.navigate(['/', 'settings', 'access-tokens', created.id], {state: {createdToken: created}});
    } finally {
      this.editFormLoading.set(false);
    }
  }

  public async deleteAccessToken(row: AccessTokenRow) {
    if (await firstValueFrom(this.overlay.confirm(`Really delete token '${row.name}'?`))) {
      try {
        await firstValueFrom(this.accessTokensService.delete(row.token.id!));
        this.accessTokens.reload();
      } catch (e) {
        const message = getFormDisplayedError(e);
        if (message) {
          this.toast.error(message);
        }
      }
    }
  }
}
