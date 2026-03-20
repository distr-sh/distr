// YC Badge
import ycBadge from './yc/yc.svg';

// Distr Screenshots
import distrArtifactLicensesDark from './screenshots/distr/distr-artifact-licenses-dark.webp';
import distrArtifactLicensesLight from './screenshots/distr/distr-artifact-licenses-light.webp';
import distrArtifactsDark from './screenshots/distr/distr-artifacts-dark.webp';
import distrArtifactsLight from './screenshots/distr/distr-artifacts-light.webp';
import distrCustomerPortalArtifactsDark from './screenshots/distr/distr-customer-portal-artifacts-dark.webp';
import distrCustomerPortalArtifactsLight from './screenshots/distr/distr-customer-portal-artifacts-light.webp';
import distrCustomerPortalArtifacts from './screenshots/distr/distr-customer-portal-artifacts.webp';
import distrCustomerPortalDark from './screenshots/distr/distr-customer-portal-dark.webp';
import distrCustomerPortalLight from './screenshots/distr/distr-customer-portal-light.webp';
import distrDashboardDark from './screenshots/distr/distr-dashboard-dark.webp';
import distrDashboardLight from './screenshots/distr/distr-dashboard-light.webp';
import distrDeploymentsDark from './screenshots/distr/distr-deployments-dark.webp';
import distrDeploymentsLight from './screenshots/distr/distr-deployments-light.webp';

export const images = {
  yc: {
    badge: ycBadge,
  },
  distr: {
    deployments: {
      light: distrDeploymentsLight,
      dark: distrDeploymentsDark,
    },
    artifacts: {
      light: distrArtifactsLight,
      dark: distrArtifactsDark,
    },
    artifactLicenses: {
      light: distrArtifactLicensesLight,
      dark: distrArtifactLicensesDark,
    },
    customerPortalArtifacts: {
      light: distrCustomerPortalArtifactsLight,
      dark: distrCustomerPortalArtifactsDark,
      default: distrCustomerPortalArtifacts,
    },
    dashboard: {
      light: distrDashboardLight,
      dark: distrDashboardDark,
    },
    customerPortal: {
      light: distrCustomerPortalLight,
      dark: distrCustomerPortalDark,
    },
  },
};
