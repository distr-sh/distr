import {Component, inject} from '@angular/core';
import {firstValueFrom} from 'rxjs';
import {AuthService} from '../services/auth.service';
import {ToastService} from '../services/toast.service';

@Component({
  selector: 'app-verify',
  templateUrl: './verify.component.html',
  imports: [],
})
export class VerifyComponent {
  private readonly toast = inject(ToastService);
  private readonly auth = inject(AuthService);
  public requestMailEnabled = true;

  public async requestMail() {
    this.requestMailEnabled = false;
    try {
      await firstValueFrom(this.auth.requestEmailVerification());
      this.toast.success('Verification email has been sent. Check your inbox.');
    } catch (e) {
      this.requestMailEnabled = true;
    }
  }

  public async logoutAndRedirectToLogin() {
    await firstValueFrom(this.auth.logout());
    location.assign('/login');
  }
}
