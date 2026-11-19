import {defineRoutedTabs} from '../navigation/tabs';

export type SettingsTabId = 'general' | 'team' | 'secrets' | 'domains-and-sign-in' | 'subscription' | 'billing';

export const organizationSettingsTabs = defineRoutedTabs<SettingsTabId>('Organization Settings', [
  {id: 'general', label: 'General', visible: (c) => c.user.isAdminOrSuperAdmin},
  {id: 'team', label: 'Team', visible: (c) => c.user.isAdminOrSuperAdmin},
  {id: 'secrets', label: 'Secrets'},
  {
    id: 'domains-and-sign-in',
    label: 'Domains & Sign-in',
    visible: (c) => c.user.isAdminOrSuperAdmin,
    locked: {requiresAnyOf: ['custom_oidc_providers', 'custom_domains', 'custom_emails'], upgradePlan: 'business'},
  },
  {
    id: 'subscription',
    label: 'Subscription',
    visible: (c) => c.user.isAdminOrSuperAdmin && c.edition !== 'community',
  },
  {id: 'billing', label: 'Billing', visible: (c) => c.user.isAdminOrSuperAdmin && c.features.has('vendor_billing')},
]);
