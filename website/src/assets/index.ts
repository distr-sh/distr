// YC Badge
import ycBadge from './yc/yc.svg';

// Distr Screenshots
import distrArtifactLicensesDark from './screenshots/distr/distr-artifact-licenses-dark.webp';
import distrArtifactLicensesLight from './screenshots/distr/distr-artifact-licenses-light.webp';
import distrArtifactsDark from './screenshots/distr/distr-artifacts-dark.webp';
import distrArtifactsLight from './screenshots/distr/distr-artifacts-light.webp';
import distrCustomerPortalDark from './screenshots/distr/distr-customer-portal-dark.webp';
import distrCustomerPortalLight from './screenshots/distr/distr-customer-portal-light.webp';
import distrDashboardDark from './screenshots/distr/distr-dashboard-dark.webp';
import distrDashboardLight from './screenshots/distr/distr-dashboard-light.webp';
import distrLogViewerDark from './screenshots/distr/distr-log-viewer-dark.webp';
import distrLogViewerLight from './screenshots/distr/distr-log-viewer-light.webp';
import distrSecurityDark from './screenshots/distr/distr-security-dark.webp';
import distrSecurityLight from './screenshots/distr/distr-security-light.webp';

export const images = {
  yc: {
    badge: ycBadge,
  },
  distr: {
    artifacts: {
      light: distrArtifactsLight,
      dark: distrArtifactsDark,
    },
    artifactLicenses: {
      light: distrArtifactLicensesLight,
      dark: distrArtifactLicensesDark,
    },
    dashboard: {
      light: distrDashboardLight,
      dark: distrDashboardDark,
    },
    customerPortal: {
      light: distrCustomerPortalLight,
      dark: distrCustomerPortalDark,
    },
    logViewer: {
      light: distrLogViewerLight,
      dark: distrLogViewerDark,
    },
    security: {
      light: distrSecurityLight,
      dark: distrSecurityDark,
    },
  },
};
