export type Edition = 'community' | 'enterprise';

export interface Environment {
  production: boolean;
  edition: Edition;
}

export interface RemoteEnvironment {
  readonly sentryDsn?: string;
  readonly sentryEnvironment?: string;
  readonly sentryTraceSampleRate?: number;
  readonly posthogToken?: string;
  readonly posthogApiHost?: string;
  readonly posthogUiHost?: string;
  readonly registryHost: string;
  readonly customDomainAppCnameTarget?: string;
  readonly customDomainRegistryCnameTarget?: string;
}
