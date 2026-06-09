import {ChangeDetectionStrategy, Component, inject} from '@angular/core';
import {firstValueFrom} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {AuthService} from '../../services/auth.service';
import {OrganizationService} from '../../services/organization.service';
import {OverlayService} from '../../services/overlay.service';
import {ToastService} from '../../services/toast.service';

@Component({
  selector: 'app-delete-organization',
  changeDetection: ChangeDetectionStrategy.Eager,
  templateUrl: './delete-organization.component.html',
})
export class DeleteOrganizationComponent {
  private readonly organizationService = inject(OrganizationService);
  private readonly toast = inject(ToastService);
  private readonly overlayService = inject(OverlayService);
  private readonly auth = inject(AuthService);

  async deleteOrganization() {
    try {
      const organization = await firstValueFrom(this.organizationService.get());
      if (
        await firstValueFrom(
          this.overlayService.confirm({
            message: {
              message:
                'Are you sure you want to delete this organization? ' +
                'Afterwards, all user sessions (including the current one) will be invalidated ' +
                'and users will be redirected to the login page.',
              alert: {
                type: 'danger',
                message: 'This is a destructive action and cannot be undone!',
              },
            },
            requiredConfirmInputText: `DELETE ${organization.name.toUpperCase()}`,
          })
        )
      ) {
        const email = this.auth.getClaims()?.email;
        await firstValueFrom(this.organizationService.delete());
        await firstValueFrom(this.auth.logout());
        location.assign(`/login?email=${encodeURIComponent(email ?? '')}`);
      }
    } catch (e) {
      const msg = getFormDisplayedError(e);
      if (msg) {
        this.toast.error(msg);
      }
    }
  }
}
