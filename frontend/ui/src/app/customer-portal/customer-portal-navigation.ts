import {defineRoutedTabs} from '../navigation/tabs';

export type CustomerPortalTabId = 'branding' | 'domain-and-sign-in';

export const customerPortalTabs = defineRoutedTabs<CustomerPortalTabId>('Customer Portal', [
  {id: 'branding', label: 'Branding'},
  {
    id: 'domain-and-sign-in',
    label: 'Domain & Sign-in',
    visible: (c) => c.user.isAdminOrSuperAdmin,
    locked: {
      plan: 'business',
      when: (c) => !c.features.has('custom_oidc_providers') && !c.features.has('custom_domains'),
    },
  },
]);
