import {OVERLAY_DEFAULT_CONFIG} from '@angular/cdk/overlay';
import {provideHttpClient, withInterceptors} from '@angular/common/http';
import {
  ApplicationConfig,
  ErrorHandler,
  inject,
  provideAppInitializer,
  provideZoneChangeDetection,
} from '@angular/core';
import {provideRouter} from '@angular/router';
import * as Sentry from '@sentry/angular';
import {routes} from './app.routes';
import {tokenInterceptor} from './services/auth.service';
import {errorToastInterceptor} from './services/error-toast.interceptor';

export const appConfig: ApplicationConfig = {
  providers: [
    {
      provide: ErrorHandler,
      useValue: Sentry.createErrorHandler(),
    },
    provideZoneChangeDetection({eventCoalescing: true}),
    provideRouter(routes),
    provideHttpClient(withInterceptors([tokenInterceptor, errorToastInterceptor])),
    provideAppInitializer(async () => inject(Sentry.TraceService)),
    {provide: OVERLAY_DEFAULT_CONFIG, useValue: {usePopover: false}},
  ],
};
