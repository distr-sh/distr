import {defineRoutedTabs} from '../navigation/tabs';

export type SettingsTabId = 'general' | 'team' | 'secrets' | 'identity-provider' | 'subscription' | 'billing';

export const organizationSettingsTabs = defineRoutedTabs<SettingsTabId>('Organization Settings', [
  {id: 'general', label: 'General', visible: (c) => c.user.isAdminOrSuperAdmin},
  {id: 'team', label: 'Team', visible: (c) => c.user.isAdminOrSuperAdmin},
  {id: 'secrets', label: 'Secrets'},
  {
    id: 'identity-provider',
    label: 'Domains & Sign-in',
    visible: (c) => c.user.isAdminOrSuperAdmin,
    locked: {
      plan: 'business',
      when: (c) =>
        !c.features.has('custom_oidc_providers') &&
        !c.features.has('custom_domains') &&
        !c.features.has('custom_emails'),
    },
  },
  {
    id: 'subscription',
    label: 'Subscription',
    visible: (c) => c.user.isAdminOrSuperAdmin && c.edition !== 'community',
  },
  {id: 'billing', label: 'Billing', visible: (c) => c.user.isAdminOrSuperAdmin && c.features.has('vendor_billing')},
]);
