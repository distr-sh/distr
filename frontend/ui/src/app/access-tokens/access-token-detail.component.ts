import {OverlayModule} from '@angular/cdk/overlay';
import {DatePipe} from '@angular/common';
import {Component, computed, effect, ElementRef, inject, signal, TemplateRef, viewChild} from '@angular/core';
import {rxResource, takeUntilDestroyed, toSignal} from '@angular/core/rxjs-interop';
import {FormControl, FormGroup, ReactiveFormsModule} from '@angular/forms';
import {ActivatedRoute, RouterLink} from '@angular/router';
import {UserRole} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {faChevronDown, faKey, faPlus, faTrash, faTriangleExclamation, faXmark} from '@fortawesome/free-solid-svg-icons';
import dayjs from 'dayjs';
import {catchError, concatMap, firstValueFrom, Observable, of, tap} from 'rxjs';
import {isExpired, RelativeDatePipe} from '../../util/dates';
import {getFormDisplayedError} from '../../util/errors';
import {USER_ROLE_LABELS} from '../../util/user-role';
import {ClipComponent} from '../components/clip.component';
import {CreatedAccessTokenComponent} from '../components/created-access-token.component';
import {
  EXPIRES_AT_DATE_FORMAT,
  ExpiresAtPickerComponent,
} from '../components/expires-at-picker/expires-at-picker.component';
import {InlineEditComponent} from '../components/inline-edit.component';
import {PageComponent} from '../components/page.component';
import {UserRoleSelectComponent} from '../components/user-role-select.component';
import {AccessTokensService} from '../services/access-tokens.service';
import {AuthService} from '../services/auth.service';
import {CreatedAccessTokenStore} from '../services/created-access-token.service';
import {DialogRef, OverlayService} from '../services/overlay.service';
import {ToastService} from '../services/toast.service';
import {AccessToken, AccessTokenSecretSlot, AccessTokenWithKey, PatchAccessTokenRequest} from '../types/access-token';
import {accessTokenName} from './access-token-name';

@Component({
  selector: 'app-access-token-detail',
  imports: [
    FaIconComponent,
    DatePipe,
    OverlayModule,
    ReactiveFormsModule,
    RouterLink,
    RelativeDatePipe,
    ClipComponent,
    CreatedAccessTokenComponent,
    ExpiresAtPickerComponent,
    InlineEditComponent,
    PageComponent,
    UserRoleSelectComponent,
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

  protected readonly expired = computed(() => {
    const token = this.token();
    return token !== undefined && isExpired(token);
  });
  protected readonly secrets = computed(() =>
    (this.token()?.secrets ?? []).map((secret) => ({...secret, expired: isExpired(secret)}))
  );
  // A token without secrets predates them and is the whole credential on its own, so giving it one
  // invalidates the token that is in circulation.
  protected readonly legacy = computed(() => this.secrets().length === 0);
  protected readonly canCreateSecret = computed(() => this.secrets().length < 2);
  protected readonly canDeleteSecret = computed(() => this.secrets().length > 1);

  protected readonly currentUserRole = computed<UserRole | undefined>(() => this.auth.getClaims()?.role);
  protected readonly inheritOptionLabel = computed(() => {
    const role = this.currentUserRole();
    return role ? `Inherit (${USER_ROLE_LABELS[role]})` : 'Inherit from my role';
  });
  protected readonly createdToken = signal<AccessTokenWithKey | null>(null);
  protected readonly savingLabel = signal(false);

  protected readonly roleControl = new FormControl<UserRole | undefined>(undefined);
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
    // Filling the control from the loaded token must not emit, or the value that just arrived from
    // the server would immediately be sent back to it.
    effect(() => this.roleControl.setValue(this.token()?.userRole, {emitEvent: false}));
    this.roleControl.valueChanges
      .pipe(
        // The requests are sequenced because an older one that finishes last would put the value it
        // was started with back, and unsubscribing does not undo one the server has already accepted.
        concatMap((userRole) => this.patch({userRole: userRole ?? null})),
        takeUntilDestroyed()
      )
      .subscribe();
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
      this.toast.success('secret created');
      this.accessTokens.reload();
    } catch (e) {
      this.showError(e);
    } finally {
      this.secretFormLoading.set(false);
    }
  }

  public async deleteSecret(slot: AccessTokenSecretSlot) {
    const confirmation =
      `Really delete secret ${slot} of token '${this.name()}'? ` +
      'Everything that still authenticates with it stops working.';
    if (await firstValueFrom(this.overlay.confirm(confirmation))) {
      try {
        await firstValueFrom(this.accessTokensService.deleteSecret(this.tokenId()!, slot));
        this.toast.success('secret deleted');
        // The token on screen may be the one this secret belonged to, and it no longer works.
        this.createdToken.set(null);
        this.accessTokens.reload();
      } catch (e) {
        this.showError(e);
      }
    }
  }

  private showError(e: unknown) {
    const message = getFormDisplayedError(e);
    if (message) {
      this.toast.error(message);
    }
  }
}
