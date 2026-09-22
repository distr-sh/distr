import {DeploymentStatusType} from '@distr-sh/distr-sdk';
import {never} from '../../util/exhaust';

export const staleDeploymentStatusBadgeClass =
  'bg-yellow-100 text-yellow-800 border-yellow-400 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800';

export function deploymentStatusBadgeClass(status: DeploymentStatusType): string {
  switch (status) {
    case 'healthy':
    case 'running':
      return 'bg-green-100 text-green-800 border-green-400 dark:bg-green-900 dark:text-green-300 dark:border-green-800';
    case 'progressing':
      return 'bg-blue-100 text-blue-800 border-blue-400 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800';
    case 'error':
      return 'bg-red-100 text-red-800 border-red-400 dark:bg-red-900 dark:text-red-300 dark:border-red-800';
    default:
      return never(status);
  }
}
