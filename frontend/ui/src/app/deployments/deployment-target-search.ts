import {DeploymentTarget} from '@distr-sh/distr-sdk';

/**
 * The texts a search for a deployment target matches, for the deployments page and the command palette. A target
 * without a customer belongs to the vendor, so it matches "Internal" and the vendor's name.
 */
export function deploymentTargetSearchTexts(
  target: DeploymentTarget,
  organizationName: string | undefined
): (string | undefined)[] {
  return [
    target.name,
    ...(target.customerOrganization ? [target.customerOrganization.name] : ['Internal', organizationName]),
    ...target.deployments.map((deployment) => deployment.application.name),
  ];
}

export function matchesSearch(texts: (string | undefined)[], query: string): boolean {
  return texts.some((text) => text?.toLowerCase().includes(query) ?? false);
}
