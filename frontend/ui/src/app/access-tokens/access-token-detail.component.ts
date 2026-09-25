import {OverlayModule} from '@angular/cdk/overlay';
import {DatePipe} from '@angular/common';
import {Component, computed, effect, ElementRef, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {rxResource, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {ActivatedRoute, Router, RouterLink} from '@angular/router';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faKey, faPlus, faTrash, faTriangleExclamation, faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {catchError, firstValueFrom, Observable, of, tap} from 'rxjs';
import {isExpired, RelativeDatePipe} from '../../util/dates';
import {getFormDisplayedError} from '../../util/errors';
import {USER_ROLE_LABELS} from '../../util/user-role';
import {CreatedAccessTokenComponent} from '../components/created-access-token.component';
import {
  EXPIRES_AT_DATE_FORMAT,
  ExpiresAtPickerComponent,
} from '../components/expires-at-picker/expires-at-picker.component';
import {InlineEditComponent} from '../components/inline-edit.component';
import {PageComponent} from '../components/page.component';
import {AccessTokensService} from '../services/access-tokens.service';
import {AuthService} from '../services/auth.service';
import {CreatedAccessTokenStore} from '../services/created-access-token.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {AccessToken, AccessTokenSecretSlot, AccessTokenWithKey, PatchAccessTokenRequest} from '../types/access-token';
import {accessTokenName} from './access-token-name';

// One credential of an access token: a secret in one of the two slots, or the key itself while it
// still authenticates on its own. The prefix is per credential, since a legacy key is shown in the
// hex encoding it was issued in and a secret in the one the current format uses.
interface CredentialRow {
  slot?: AccessTokenSecretSlot;
  label: string;
  keyId: string;
  createdAt: string;
  expiresAt?: string;
  lastUsedAt?: string;
  expired: boolean;
}

@Component({
  selector: 'app-access-token-detail',
  imports: [
    FaIconComponent,
    DatePipe,
    OverlayModule,
    ReactiveFormsModule,
    RouterLink,
    RelativeDatePipe,
    CreatedAccessTokenComponent,
    ExpiresAtPickerComponent,
    InlineEditComponent,
    PageComponent,
  ],
  templateUrl: './access-token-detail.component.html',
})
export class AccessTokenDetailComponent {
  protected readonly faChevronDown = faChevronDown;
  protected readonly faKey = faKey;
  protected readonly faPlus = faPlus;
  protected readonly faTrash = faTrash;
  protected readonly faTriangleExclamation = faTriangleExclamation;
  protected readonly faXmark = faXmark;

  private readonly accessTokensService = inject(AccessTokensService);
  private readonly auth = inject(AuthService);
  private readonly overlay = inject(OverlayService);
  private readonly toast = inject(ToastService);
  private readonly createdTokens = inject(CreatedAccessTokenStore);
  private readonly router = inject(Router);
  private readonly routeParams = toSignal(inject(ActivatedRoute).params);

  private readonly accessTokens = rxResource({stream: () => this.accessTokensService.list()});

  protected readonly loading = computed(() => this.accessTokens.isLoading());
  protected readonly tokenId = computed(() => this.routeParams()?.['accessTokenId'] as string | undefined);
  protected readonly token = computed(() => (this.accessTokens.value() ?? []).find((t) => t.id === this.tokenId()));
  protected readonly name = computed(() => {
    const token = this.token();
    return token ? accessTokenName(token) : '';
  });
  protected readonly siblings = computed(() =>
    (this.accessTokens.value() ?? []).map((token) => ({id: token.id!, name: accessTokenName(token)}))
  );

  protected readonly credentials = computed<CredentialRow[]>(() => {
    const token = this.token();
    if (!token) {
      return [];
    }
    const rows: CredentialRow[] = [];
    if (token.legacyKey) {
      rows.push({
        label: 'Legacy',
        keyId: token.legacyKey.keyId,
        createdAt: token.legacyKey.createdAt,
        expiresAt: token.legacyKey.expiresAt,
        lastUsedAt: token.legacyKey.lastUsedAt,
        expired: isExpired(token.legacyKey),
      });
    }
    for (const secret of token.secrets) {
      rows.push({
        slot: secret.slot,
        label: `Token ${secret.slot}`,
        keyId: token.keyId,
        createdAt: secret.createdAt,
        expiresAt: secret.expiresAt,
        lastUsedAt: secret.lastUsedAt,
        expired: isExpired(secret),
      });
    }
    return rows;
  });
  // The key of a token issued before secrets existed keeps authenticating until it is deleted, so
  // adding a secret migrates such a token without cutting off whatever holds the old one.
  protected readonly legacy = computed(() => this.token()?.legacyKey !== undefined);
  protected readonly canCreateSecret = computed(() => this.credentials().length < 2);
  protected readonly canDeleteCredential = computed(() => this.credentials().length > 1);
  private readonly expired = computed(() => this.credentials().every((credential) => credential.expired));

  // A token that was created without a role of its own acts under the role its owner has, which
  // is re-read on every request, so it follows them when they are promoted or demoted.
  protected readonly roleLabel = computed(() => {
    const tokenRole = this.token()?.userRole;
    if (tokenRole) {
      return USER_ROLE_LABELS[tokenRole];
    }
    const ownRole = this.auth.getClaims()?.role;
    return ownRole ? `Inherited (${USER_ROLE_LABELS[ownRole]})` : 'Inherited';
  });
  protected readonly createdToken = signal<AccessTokenWithKey | null>(null);
  protected readonly savingLabel = signal(false);

  protected readonly secretForm = new FormGroup({expiresAt: new FormControl('', {nonNullable: true})});
  protected readonly secretFormLoading = signal(false);
  private secretModal: DialogRef<void> | null = null;

  constructor() {
    // The router reuses this component for a sibling token, so the token a create handed over
    // belongs to the id that was current when it was stored and to no other.
    effect(() => {
      const id = this.tokenId();
      this.createdToken.set(id ? this.createdTokens.take(id) : null);
    });
  }

  protected readonly dropdownTriggerButton = viewChild.required<ElementRef<HTMLElement>>('dropdownTriggerButton');
  protected readonly breadcrumbDropdown = signal(false);
  protected readonly breadcrumbDropdownWidth = signal(0);

  protected toggleBreadcrumbDropdown() {
    this.breadcrumbDropdown.update((open) => !open);
    if (this.breadcrumbDropdown()) {
      this.breadcrumbDropdownWidth.set(this.dropdownTriggerButton().nativeElement.getBoundingClientRect().width);
    }
  }

  public async saveLabel(label: string) {
    this.savingLabel.set(true);
    try {
      await firstValueFrom(this.patch({label}));
    } finally {
      this.savingLabel.set(false);
    }
  }

  private patch(request: PatchAccessTokenRequest): Observable<AccessToken | null> {
    return this.accessTokensService.patch(this.tokenId()!, request).pipe(
      catchError((e) => {
        this.showError(e);
        return of(null);
      }),
      tap((updated) => {
        if (updated) {
          this.toast.success('token updated');
        }
        // Reload either way, so that a rejected change is replaced by what the server still has
        // instead of staying on screen.
        this.accessTokens.reload();
      })
    );
  }

  public openSecretModal(template: TemplateRef<unknown>) {
    this.secretForm.reset({expiresAt: dayjs().add(30, 'day').format(EXPIRES_AT_DATE_FORMAT)});
    this.secretModal = this.overlay.showModal(template);
  }

  public closeSecretModal() {
    this.secretModal?.dismiss();
  }

  public async createSecret() {
    this.secretFormLoading.set(true);
    const {expiresAt} = this.secretForm.value;
    try {
      const created = await firstValueFrom(
        this.accessTokensService.createSecret(this.tokenId()!, {
          // The picker works in local dates, and new Date() would read one as UTC midnight, which
          // moves the day for everyone west of it.
          expiresAt: expiresAt ? dayjs(expiresAt).toDate() : undefined,
        })
      );
      this.closeSecretModal();
      this.createdToken.set(created);
      this.toast.success('token added');
      this.accessTokens.reload();
    } catch (e) {
      this.showError(e);
    } finally {
      this.secretFormLoading.set(false);
    }
  }

  public async deleteAccessToken() {
    if (!this.expired() && !(await firstValueFrom(this.overlay.confirm(`Really delete token '${this.name()}'?`)))) {
      return;
    }
    try {
      await firstValueFrom(this.accessTokensService.delete(this.tokenId()!));
      this.toast.success('Access Token deleted');
      await this.router.navigate(['/', 'settings', 'access-tokens']);
    } catch (e) {
      this.showError(e);
    }
  }

  public async deleteCredential(credential: CredentialRow) {
    if (!credential.expired) {
      const confirmation =
        `Really delete ${credential.slot ? `token ${credential.slot}` : 'the legacy token'} of ` +
        `'${this.name()}'? Everything that still authenticates with it stops working.`;
      if (!(await firstValueFrom(this.overlay.confirm(confirmation)))) {
        return;
      }
    }
    try {
      const id = this.tokenId()!;
      await firstValueFrom(
        credential.slot
          ? this.accessTokensService.deleteSecret(id, credential.slot)
          : this.accessTokensService.deleteLegacyKey(id)
      );
      this.toast.success('token deleted');
      // The token on screen may be the one that was deleted, and it no longer works.
      this.createdToken.set(null);
      this.accessTokens.reload();
    } catch (e) {
      this.showError(e);
    }
  }

  private showError(e: unknown) {
    const message = getFormDisplayedError(e);
    if (message) {
      this.toast.error(message);
    }
  }
}
