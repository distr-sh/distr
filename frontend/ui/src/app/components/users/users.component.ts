import {AsyncPipe, DatePipe} from '@angular/common';
import {Component, computed, inject, input, output, signal, TemplateRef, viewChild} from '@angular/core';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';
import {FormBuilder, ReactiveFormsModule, Validators} from '@angular/forms';
import {UserAccountWithRole, UserRole} from '@distr-sh/distr-sdk';
import {FaIconComponent} from '@fortawesome/angular-fontawesome';
import {
  faBox,
  faCheck,
  faCircleExclamation,
  faClipboard,
  faMagnifyingGlass,
  faPen,
  faPlus,
  faRepeat,
  faTrash,
  faUserCircle,
  faXmark,
} from '@fortawesome/free-solid-svg-icons';
import {catchError, filter, firstValueFrom, NEVER, switchMap, tap} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {filteredByFormControl} from '../../../util/filter';
import {SecureImagePipe} from '../../../util/secureImage';
import {modalFlyInOut} from '../../animations/modal';
import {AutotrimDirective} from '../../directives/autotrim.directive';
import {RequireVendorDirective} from '../../directives/required-role.directive';
import {AuthService} from '../../services/auth.service';
import {ImageUploadService} from '../../services/image-upload.service';
import {OrganizationService} from '../../services/organization.service';
import {DialogRef, OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';
import {UsersService} from '../../services/users.service';
import {QuotaLimitComponent} from '../quota-limit.component';

@Component({
  selector: 'app-users',
  imports: [
    FaIconComponent,
    AsyncPipe,
    DatePipe,
    ReactiveFormsModule,
    RequireVendorDirective,
    AutotrimDirective,
    SecureImagePipe,
    QuotaLimitComponent,
  ],
  templateUrl: './users.component.html',
  animations: [modalFlyInOut],
})
export class UsersComponent {
  public readonly users = input.required<UserAccountWithRole[]>();
  public readonly customerOrganizationId = input<string>();
  public readonly refresh = output<void>();

  private readonly toast = inject(ToastService);
  private readonly usersService = inject(UsersService);
  private readonly organizationService = inject(OrganizationService);
  private readonly overlay = inject(OverlayService);
  private readonly imageUploadService = inject(ImageUploadService);
  protected readonly auth = inject(AuthService);
  private readonly fb = inject(FormBuilder).nonNullable;

  protected readonly faBox = faBox;
  protected readonly faCheck = faCheck;
  protected readonly faCircleExclamation = faCircleExclamation;
  protected readonly faClipboard = faClipboard;
  protected readonly faMagnifyingGlass = faMagnifyingGlass;
  protected readonly faPen = faPen;
  protected readonly faPlus = faPlus;
  protected readonly faRepeat = faRepeat;
  protected readonly faTrash = faTrash;
  protected readonly faUserCircle = faUserCircle;
  protected readonly faXmark = faXmark;

  protected readonly filterForm = this.fb.group({
    search: this.fb.control(''),
  });

  protected readonly users$ = filteredByFormControl(
    toObservable(this.users),
    this.filterForm.controls.search,
    (it: UserAccountWithRole, search: string) =>
      !search ||
      (it.name || '').toLowerCase().includes(search.toLowerCase()) ||
      (it.email || '').toLowerCase().includes(search.toLowerCase())
  );

  private readonly inviteUserDialog = viewChild.required<TemplateRef<unknown>>('inviteUserDialog');
  private modalRef?: DialogRef;
  protected readonly inviteFormLoading = signal(false);
  protected readonly inviteForm = this.fb.group({
    email: this.fb.control('', [Validators.required, Validators.email]),
    name: this.fb.control<string | undefined>(undefined),
    userRole: this.fb.control<UserRole>('admin', Validators.required),
  });
  protected inviteUrl: string | null = null;

  protected readonly organization = toSignal(this.organizationService.get());

  protected readonly limit = computed(() => {
    const org = this.organization();
    return !org
      ? undefined
      : this.auth.isVendor() && this.customerOrganizationId() === undefined
        ? org.subscriptionUserAccountQuantity
        : org.subscriptionLimits.maxUsersPerCustomerOrganization;
  });

  protected readonly isProSubscription = computed(() => {
    const subscriptionType = this.organization()?.subscriptionType;
    return subscriptionType && ['trial', 'pro', 'enterprise'].includes(subscriptionType);
  });

  protected readonly editNameUserId = signal<string | null>(null);
  protected readonly editNameFormLoading = signal(false);
  protected readonly editNameForm = this.fb.group({
    name: this.fb.control(''),
  });

  protected readonly editRoleUserId = signal<string | null>(null);
  protected readonly editRoleFormLoading = signal(false);
  protected readonly editRoleForm = this.fb.group({
    userRole: this.fb.control<UserRole>('admin', Validators.required),
  });

  public showInviteDialog(reset?: boolean): void {
    this.closeInviteDialog(reset);
    this.modalRef = this.overlay.showModal(this.inviteUserDialog());
  }

  protected editUserName(user: UserAccountWithRole): void {
    if (!user.id) {
      return;
    }
    this.editNameFormLoading.set(false);
    this.editNameUserId.set(user.id);
    this.editRoleUserId.set(null);
    this.editNameForm.reset(user);
  }

  protected async submitEditUserNameForm(): Promise<void> {
    this.editNameForm.markAllAsTouched();

    const userId = this.editNameUserId();
    const name = this.editNameForm.value.name;
    if (!userId || !name) {
      return;
    }

    if (this.editNameForm.valid) {
      this.editNameFormLoading.set(true);
      try {
        await firstValueFrom(this.usersService.patchUserAccount(userId, {name}));
        this.editNameUserId.set(null);
        this.editNameForm.reset();
        this.toast.success('User has been updated');
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.editNameFormLoading.set(false);
      }
    }
  }

  protected editUserRole(user: UserAccountWithRole): void {
    if (!user.id) {
      return;
    }
    this.editRoleFormLoading.set(false);
    this.editRoleUserId.set(user.id);
    this.editNameUserId.set(null);
    this.editRoleForm.reset(user);
  }

  protected async submitEditUserRoleForm(): Promise<void> {
    this.editRoleForm.markAllAsTouched();

    const userId = this.editRoleUserId();
    const userRole = this.editRoleForm.value.userRole;
    if (!userId || !userRole) {
      return;
    }

    if (this.editRoleForm.valid) {
      this.editRoleFormLoading.set(true);
      try {
        await firstValueFrom(this.usersService.patchUserAccount(userId, {userRole}));
        this.editRoleUserId.set(null);
        this.editRoleForm.reset();
        this.toast.success('User role has been updated');
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.editRoleFormLoading.set(false);
      }
    }
  }

  public async submitInviteForm(): Promise<void> {
    this.inviteForm.markAllAsTouched();
    if (this.inviteForm.valid) {
      this.inviteFormLoading.set(true);
      try {
        const result = await firstValueFrom(
          this.usersService.addUser({
            email: this.inviteForm.value.email!,
            name: this.inviteForm.value.name || undefined,
            userRole: this.inviteForm.value.userRole ?? 'admin',
            customerOrganizationId: this.customerOrganizationId(),
          })
        );
        this.inviteUrl = result.inviteUrl;
        if (!this.inviteUrl) {
          this.toast.success(
            `${result.user.customerOrganizationId === undefined ? 'User' : 'Customer'} has been added to the organization`
          );
          this.closeInviteDialog();
        }
        this.refresh.emit();
      } catch (e) {
        const msg = getFormDisplayedError(e);
        if (msg) {
          this.toast.error(msg);
        }
      } finally {
        this.inviteFormLoading.set(false);
      }
    }
  }

  public async uploadImage(data: UserAccountWithRole) {
    const fileId = await firstValueFrom(
      this.imageUploadService.showDialog({imageUrl: data.imageUrl, scope: 'platform'})
    );
    if (!fileId || data.imageUrl?.includes(fileId)) {
      return;
    }
    await firstValueFrom(this.usersService.patchImage(data.id!, fileId));
  }

  protected async resendInvitation(user: UserAccountWithRole) {
    try {
      const result = await firstValueFrom(this.usersService.resendInvitation(user));
      this.inviteUrl = result.inviteUrl;
      if (!this.inviteUrl) {
        this.toast.success(`Invitation has been resent to ${user.email}`);
      } else {
        this.showInviteDialog(false);
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }

  public async deleteUser(user: UserAccountWithRole): Promise<void> {
    this.overlay
      .confirm(`This will remove ${user.name ?? user.email} from your organization. Are you sure?`)
      .pipe(
        filter((result) => result === true),
        switchMap(() => this.usersService.delete(user)),
        catchError((e) => {
          const msg = getFormDisplayedError(e);
          if (msg) {
            this.toast.error(msg);
          }
          return NEVER;
        }),
        tap(() => this.refresh.emit())
      )
      .subscribe();
  }

  public closeInviteDialog(reset: boolean = true): void {
    this.modalRef?.close();

    if (reset) {
      this.inviteUrl = null;
      this.inviteForm.reset();
    }
  }

  public copyInviteUrl(): void {
    if (this.inviteUrl) {
      navigator.clipboard.writeText(this.inviteUrl);
      this.toast.success('Invite URL has been copied');
    }
  }
}
