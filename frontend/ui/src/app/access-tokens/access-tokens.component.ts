import {OverlayModule} from '@angular/cdk/overlay';
import {DatePipe} from '@angular/common';
import {Component, computed, inject, signal, TemplateRef} from '@angular/core';
import {rxResource, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {Router, RouterLink} from '@angular/router';
import {UserRole} from '@distr-sh/distr-sdk';
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
import {CreatedAccessTokenStore} from '../services/created-access-token.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {AccessToken, CreateAccessTokenRequest} from '../types/access-token';
import {accessTokenName} from './access-token-name';

interface AccessTokenRow {
  token: AccessToken;
  name: string;
  // Everything the filter matches against: the label, and both encodings of the key plus the id, so
  // that a token pasted from elsewhere finds its row whichever format it is in.
  search: string;
  // An access token is spent once every credential that could authenticate it has expired.
  expired: boolean;
  // Whether the key still authenticates on its own, which is the format that predates secrets.
  legacy: boolean;
  credentials: {expiresAt?: string; expired: boolean}[];
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
  private readonly createdTokens = inject(CreatedAccessTokenStore);

  private readonly accessTokens = rxResource({stream: () => this.accessTokensService.list()});

  private readonly rows = computed<AccessTokenRow[]>(() =>
    (this.accessTokens.value() ?? []).map((token) => {
      const credentials = [...(token.legacyKey ? [token.legacyKey] : []), ...token.secrets].map((credential) => ({
        expiresAt: credential.expiresAt,
        expired: isExpired(credential),
      }));
      return {
        token,
        name: accessTokenName(token),
        search: `${token.label ?? ''} ${token.keyId} ${token.legacyKey?.keyId ?? ''} ${token.id}`.toLowerCase(),
        expired: credentials.every((credential) => credential.expired),
        legacy: token.legacyKey !== undefined,
        credentials,
      };
    })
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
      // The picker works in local dates, and new Date() would read one as UTC midnight, which
      // moves the day for everyone west of it.
      request.expiresAt = dayjs(this.editForm.value.expiresAt).toDate();
    }
    if (this.editForm.value.userRole) {
      request.userRole = this.editForm.value.userRole;
    }
    try {
      const created = await firstValueFrom(this.accessTokensService.create(request));
      this.toast.success('token created');
      this.hideDrawer();
      // The new token is handed over out of band because the server never returns it again.
      this.createdTokens.put(created);
      await this.router.navigate(['/', 'settings', 'access-tokens', created.id]);
    } finally {
      this.editFormLoading.set(false);
    }
  }

  public async deleteAccessToken(row: AccessTokenRow) {
    if (!row.expired && !(await firstValueFrom(this.overlay.confirm(`Really delete token '${row.name}'?`)))) {
      return;
    }
    try {
      await firstValueFrom(this.accessTokensService.delete(row.token.id!));
      this.toast.success('Access Token deleted');
      this.accessTokens.reload();
    } catch (e) {
      const message = getFormDisplayedError(e);
      if (message) {
        this.toast.error(message);
      }
    }
  }
}
