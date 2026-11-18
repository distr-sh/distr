import {defineRoutedTabs} from '../navigation/tabs';

export type CustomerPortalTabId = 'branding' | 'domain-and-sign-in';

export const customerPortalTabs = defineRoutedTabs<CustomerPortalTabId>('Customer Portal', [
  {id: 'branding', label: 'Branding'},
  {
    id: 'domain-and-sign-in',
    label: 'Domain & Sign-in',
    visible: (c) => c.user.isAdminOrSuperAdmin,
    locked: {requiresAnyOf: ['custom_oidc_providers', 'custom_domains'], upgradePlan: 'business'},
  },
]);
