import { bootstrapApplication } from '@angular/platform-browser';
import * as Sentry from '@sentry/angular';
import dayjs from 'dayjs';
import duration from 'dayjs/plugin/duration';
import relativeTime from 'dayjs/plugin/relativeTime';
import utc from 'dayjs/plugin/utc';
import posthog from 'posthog-js';
import { AppComponent } from './app/app.component';
import { appConfig } from './app/app.config';
import { buildConfig } from './buildconfig';
import { environment } from './env/env';
import { getRemoteEnvironment } from './env/remote';

dayjs.extend(duration);
dayjs.extend(relativeTime);
dayjs.extend(utc);

bootstrapApplication(AppComponent, appConfig).catch((err) => console.error(err));

(async () => {
  const remoteEnvironment = await getRemoteEnvironment();

  if (remoteEnvironment.sentryDsn) {
    Sentry.init({
      enabled: environment.production,
      release: buildConfig.sentryVersion ?? buildConfig.commit,
      dsn: remoteEnvironment.sentryDsn,
      environment: remoteEnvironment.sentryEnvironment,
      integrations: [Sentry.browserTracingIntegration()],
      tracesSampleRate: remoteEnvironment.sentryTraceSampleRate ?? 1,
    });
  }

  if (remoteEnvironment.posthogToken) {
    posthog.init(remoteEnvironment.posthogToken, {
      api_host: remoteEnvironment.posthogApiHost,
      ui_host: remoteEnvironment.posthogUiHost,
      person_profiles: 'identified_only',
      session_recording: {
        maskAllInputs: false,
        maskInputOptions: {
          password: true,
        },
        maskTextSelector: '[contenteditable], [data-ph-mask-text]',
      },
      // pageview event capturing is done for Angular router events.
      // Here we prevent the window "load" event from triggering a duplicate pageview event.
      capture_pageview: false,
      before_send: [
        (cr) => {
          if (cr !== null) {
            if (cr.$set === undefined) {
              cr.$set = {};
            }
            if (cr.$set_once === undefined) {
              cr.$set_once = {};
            }
            cr.$set['version'] = buildConfig.version;
            cr.$set_once['initial_version'] = buildConfig.version;
          }
          return cr;
        },
      ],
    });
  }
})().catch((err) => console.error(err));
