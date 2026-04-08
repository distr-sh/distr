import {AsyncPipe} from '@angular/common';
import {HttpErrorResponse} from '@angular/common/http';
import {Component, inject} from '@angular/core';
import {catchError, EMPTY, map, Observable} from 'rxjs';
import {getFormDisplayedError} from '../../../util/errors';
import {InnerMarkdownDirective} from '../../directives/inner-markdown.directive';
import {OrganizationBrandingService} from '../../services/organization-branding.service';
import {ToastService} from '../../services/toast.service';

@Component({
  selector: 'app-home',
  imports: [AsyncPipe, InnerMarkdownDirective],
  templateUrl: './home.component.html',
})
export class HomeComponent {
  private readonly organizationBranding = inject(OrganizationBrandingService);
  private toast = inject(ToastService);
  readonly brandingDescription$: Observable<string | undefined> = this.organizationBranding.get().pipe(
    catchError((e) => {
      const msg = getFormDisplayedError(e);
      if (msg && e instanceof HttpErrorResponse && e.status !== 404) {
        this.toast.error(msg);
      }
      return EMPTY;
    }),
    map((b) => b.description)
  );
}
