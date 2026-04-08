import {Component, effect, inject, OnInit} from '@angular/core';
import {Event, NavigationEnd, Router, RouterOutlet} from '@angular/router';
import {FontAwesomeModule} from '@fortawesome/angular-fontawesome';
import * as Sentry from '@sentry/angular';
import posthog from 'posthog-js';
import {filter, Observable} from 'rxjs';
import {ToastContainerComponent} from './components/toast-container.component';
import {AuthService} from './services/auth.service';
import {ColorSchemeService} from './services/color-scheme.service';
import {ImageUploadService} from './services/image-upload.service';
import {OverlayService} from './services/overlay.service';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [RouterOutlet, FontAwesomeModule, ToastContainerComponent],
  providers: [OverlayService, ImageUploadService],
  template: `<router-outlet /><app-toast-container />`,
})
export class AppComponent implements OnInit {
  private readonly router = inject(Router);
  private readonly auth = inject(AuthService);
  private readonly navigationEnd$: Observable<NavigationEnd> = this.router.events.pipe(
    filter((event: Event) => event instanceof NavigationEnd)
  );

  constructor(private readonly colorSchemeService: ColorSchemeService) {
    effect(() => {
      document.body.classList.toggle('dark', this.colorSchemeService.colorScheme() === 'dark');
    });
  }

  public ngOnInit() {
    this.navigationEnd$.subscribe(() => {
      const jwtClaims = this.auth.getClaims();
      if (jwtClaims) {
        const email = jwtClaims.email;
        Sentry.setUser({email});
        posthog.setPersonProperties({email});
        posthog.group('organization', jwtClaims.org);
      }
      posthog.capture('$pageview');
    });
  }
}
