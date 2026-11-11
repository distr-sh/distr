import {CustomerOrganizationWithUsage} from '@distr-sh/distr-sdk';
import {ConfirmConfig} from '../confirm-dialog/confirm-dialog.component';

export function deleteCustomerConfirmConfig(target: CustomerOrganizationWithUsage): ConfirmConfig {
  return {
    message: {
      message: 'Are you sure you want to delete this customer?',
      alert:
        target.userCount > 0 || target.deploymentTargetCount > 0
          ? {
              type: 'warning',
              message: `Deleting this customer will also delete its associated users (${target.userCount}) and deployment targets (${target.deploymentTargetCount}) from your organization.`,
            }
          : {
              type: 'info',
              message: 'This customer has no associated users or deployment targets.',
            },
    },
    requiredConfirmInputText: target.name,
  };
}
