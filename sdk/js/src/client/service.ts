import semver from 'semver/preload';
import {
  Application,
  ApplicationVersion,
  ApplicationVersionResource,
  DeploymentTarget,
  DeploymentTargetAccessResponse,
  DeploymentTargetScope,
  DeploymentType,
  HelmChartType,
} from '../types';
import {Client, ClientConfig} from './client';
import {ConditionalPartial, defaultClientConfig} from './config';

/**
 * The strategy for determining the latest version of an application.
 * * 'semver' uses semantic versioning to determine the latest version.
 * * 'chronological' uses the creation date of the versions to determine the latest version.
 */
export type LatestVersionStrategy = 'semver' | 'chronological';

export type CreateDeploymentParams = {
  target: {
    name: string;
    type: DeploymentType;
    kubernetes?: {
      namespace: string;
      scope: DeploymentTargetScope;
    };
  };
  application: {
    id?: string;
    versionId?: string;
  };
  kubernetesDeployment?: {
    releaseName: string;
    valuesYaml?: string;
  };
};

export type CreateDeploymentResult = {
  deploymentTarget: DeploymentTarget;
  access: DeploymentTargetAccessResponse;
};

export type UpdateDeploymentParams = {
  deploymentTargetId: string;
  applicationId: string;
  applicationVersionId: string;
  kubernetesDeployment?: {
    valuesYaml?: string;
  };
};

export type UpdateAllDeploymentsResult = {
  updatedTargets: Array<{
    deploymentTargetId: string;
    deploymentTargetName: string;
    previousVersionId: string;
    newVersionId: string;
  }>;
  skippedTargets: Array<{
    deploymentTargetId: string;
    deploymentTargetName: string;
    reason: string;
  }>;
};

export type IsOutdatedResultItem = {
  deployment: {
    id: string;
    applicationId: string;
    applicationVersionId: string;
  };
  application: Application;
  newerVersions: ApplicationVersion[];
  outdated: boolean;
};

export type IsOutdatedResult = {
  deploymentTarget: DeploymentTarget;
  results: IsOutdatedResultItem[];
};

/**
 * The DistrService provides a higher-level API for the Distr API. It allows to create and update deployments, check
 * if a deployment is outdated, and get the latest version of an application according to a specified strategy.
 * Under the hood it uses the low-level {@link Client}.
 */
export class DistrService {
  private readonly client: Client;

  /**
   * Creates a new DistrService instance. A client config containing an API key must be provided, optionally the API
   * base URL can be set. Optionally, a strategy for determining the latest version of an application can be specified –
   * the default is semantic versioning.
   * @param config ClientConfig containing at least an API key and optionally an API base URL
   * @param latestVersionStrategy Strategy for determining the latest version of an application (default: 'semver')
   */
  constructor(
    config: ConditionalPartial<ClientConfig, keyof typeof defaultClientConfig>,
    private readonly latestVersionStrategy: LatestVersionStrategy = 'semver'
  ) {
    this.client = new Client(config);
  }

  /**
   * Creates a new application version for the given docker application using a Docker Compose file and an
   * optional template file.
   * @param applicationId
   * @param name Name of the new version
   * @param data
   */
  public async createDockerApplicationVersion(
    applicationId: string,
    name: string,
    data: {
      composeFile: string;
      templateFile?: string;
      linkTemplate?: string;
      resources?: ApplicationVersionResource[];
    }
  ): Promise<ApplicationVersion> {
    return this.client.createApplicationVersion(
      applicationId,
      {name, linkTemplate: data.linkTemplate ?? '', resources: data.resources},
      {
        composeFile: data.composeFile,
        templateFile: data.templateFile,
      }
    );
  }

  /**
   * Creates a new application version for the given Kubernetes application using a Helm chart.
   * @param applicationId
   * @param versionName
   * @param data
   */
  public async createKubernetesApplicationVersion(
    applicationId: string,
    versionName: string,
    data: {
      chartName?: string;
      chartVersion: string;
      chartType: HelmChartType;
      chartUrl: string;
      baseValuesFile?: string;
      templateFile?: string;
      linkTemplate?: string;
      resources?: ApplicationVersionResource[];
    }
  ): Promise<ApplicationVersion> {
    return this.client.createApplicationVersion(
      applicationId,
      {
        name: versionName,
        linkTemplate: data.linkTemplate ?? '',
        chartName: data.chartName,
        chartVersion: data.chartVersion,
        chartType: data.chartType,
        chartUrl: data.chartUrl,
        resources: data.resources,
      },
      {
        baseValuesFile: data.baseValuesFile,
        templateFile: data.templateFile,
      }
    );
  }

  /**
   * Creates a new deployment target and deploys the given application version to it.
   * * If deployment type is 'kubernetes', the namespace and scope must be provided.
   * * If deployment type is 'kubernetes', the helm release name and values YAML can be provided.
   * * If no application version ID is given, the latest version of the application will be deployed.
   * @param params
   */
  public async createDeployment(params: CreateDeploymentParams): Promise<CreateDeploymentResult> {
    const {target, application, kubernetesDeployment} = params;

    let versionId = application.versionId;
    if (!versionId) {
      if (!application.id) {
        throw new Error('application ID or version ID must be provided');
      }
      const latest = await this.getLatestVersion(application.id);
      if (!latest) {
        throw new Error('no version available for this application');
      }
      versionId = latest.id!;
    }

    const deploymentTarget = await this.client.createDeploymentTarget({
      name: target.name,
      type: target.type,
      namespace: target.kubernetes?.namespace,
      scope: target.kubernetes?.scope,
      deployments: [],
      metricsEnabled: false,
    });
    await this.client.createOrUpdateDeployment({
      deploymentTargetId: deploymentTarget.id!,
      applicationVersionId: versionId,
      releaseName: kubernetesDeployment?.releaseName,
      valuesYaml: kubernetesDeployment?.valuesYaml ? btoa(kubernetesDeployment?.valuesYaml) : undefined,
    });
    return {
      deploymentTarget: await this.client.getDeploymentTarget(deploymentTarget.id!),
      access: await this.client.createAccessForDeploymentTarget(deploymentTarget.id!),
    };
  }

  /**
   * Updates the deployment of an existing deployment target to the specified application version.
   * @param params
   */
  public async updateDeployment(params: UpdateDeploymentParams): Promise<void> {
    const {deploymentTargetId, applicationId, applicationVersionId, kubernetesDeployment} = params;

    const existing = await this.client.getDeploymentTarget(deploymentTargetId);
    const existingDeployment = existing.deployments.find((d) => d.application.id === applicationId);
    if (!existingDeployment) {
      throw new Error(`cannot update deployment, no deployment found for application ${applicationId}`);
    }
    await this.client.createOrUpdateDeployment({
      deploymentTargetId,
      deploymentId: existingDeployment.id,
      applicationVersionId,
      valuesYaml: kubernetesDeployment?.valuesYaml ? btoa(kubernetesDeployment?.valuesYaml) : undefined,
    });
  }

  /**
   * Updates all deployment targets that have the specified application deployed to the specified version.
   * Only updates deployments that are not already on the target version.
   * @param applicationId The application ID to update
   * @param applicationVersionId The target version ID to update to
   */
  public async updateAllDeployments(
    applicationId: string,
    applicationVersionId: string
  ): Promise<UpdateAllDeploymentsResult> {
    const allTargets = await this.client.getDeploymentTargets();
    const updatedTargets: UpdateAllDeploymentsResult['updatedTargets'] = [];
    const skippedTargets: UpdateAllDeploymentsResult['skippedTargets'] = [];

    for (const target of allTargets) {
      const deployment = target.deployments?.find((d) => d.application.id === applicationId);
      if (!deployment) {
        skippedTargets.push({
          deploymentTargetId: target.id!,
          deploymentTargetName: target.name,
          reason: 'Application not deployed on this target',
        });
        continue;
      }

      if (deployment.applicationVersionId === applicationVersionId) {
        skippedTargets.push({
          deploymentTargetId: target.id!,
          deploymentTargetName: target.name,
          reason: 'Already on target version',
        });
        continue;
      }

      try {
        await this.client.createOrUpdateDeployment({
          deploymentTargetId: target.id!,
          deploymentId: deployment.id,
          applicationVersionId,
        });
        updatedTargets.push({
          deploymentTargetId: target.id!,
          deploymentTargetName: target.name,
          previousVersionId: deployment.applicationVersionId!,
          newVersionId: applicationVersionId,
        });
      } catch (error) {
        skippedTargets.push({
          deploymentTargetId: target.id!,
          deploymentTargetName: target.name,
          reason: `Update failed: ${error instanceof Error ? error.message : String(error)}`,
        });
      }
    }

    return {updatedTargets, skippedTargets};
  }

  /**
   * Checks if the deployments on the given deployment target are outdated, i.e. if there is a newer version of the application available.
   * Returns results for all deployments on the target. Each result contains versions that are newer than the currently deployed one, ordered ascending.
   * @param deploymentTargetId
   */
  public async isOutdated(deploymentTargetId: string): Promise<IsOutdatedResult> {
    const existing = await this.client.getDeploymentTarget(deploymentTargetId);
    if (existing.deployments.length === 0) {
      throw new Error('nothing deployed yet');
    }
    const results: IsOutdatedResultItem[] = [];
    for (const deployment of existing.deployments) {
      const {app, newerVersions} = await this.getNewerVersions(
        deployment.application.id!,
        deployment.applicationVersionId
      );
      results.push({
        deployment: {
          id: deployment.id!,
          applicationId: deployment.application.id!,
          applicationVersionId: deployment.applicationVersionId,
        },
        application: app,
        newerVersions: newerVersions,
        outdated: newerVersions.length > 0,
      });
    }
    return {
      deploymentTarget: existing,
      results,
    };
  }

  /**
   * Returns the latest version of the given application according to the specified strategy.
   * @param appId
   */
  public async getLatestVersion(appId: string): Promise<ApplicationVersion | undefined> {
    const {newerVersions} = await this.getNewerVersions(appId);
    return newerVersions.length > 0 ? newerVersions[newerVersions.length - 1] : undefined;
  }

  /**
   * Returns the application and all versions that are newer than the given version ID. If no version ID is given,
   * all versions are considered. The versions are ordered ascending according to the given strategy.
   * @param appId
   * @param currentVersionId
   */
  public async getNewerVersions(
    appId: string,
    currentVersionId?: string
  ): Promise<{app: Application; newerVersions: ApplicationVersion[]}> {
    const app = await this.client.getApplication(appId);
    const currentVersion = (app.versions || []).find((it) => it.id === currentVersionId);
    if (!currentVersion && currentVersionId) {
      throw new Error('given version ID does not exist in this application');
    }
    const newerVersions = (app.versions || [])
      .filter((it) => {
        if (!currentVersion) {
          return true;
        }
        // surely there are fancier ways to deal with strategies but that's it for now
        switch (this.latestVersionStrategy) {
          case 'semver':
            return semver.gt(it.name!, currentVersion.name!, {loose: true});
          case 'chronological':
            return it.createdAt! > currentVersion.createdAt!; // TODO proper date handling maybe
        }
      })
      .sort((a, b) => {
        switch (this.latestVersionStrategy) {
          case 'semver':
            return semver.compare(a.name!, b.name!, {loose: true});
          case 'chronological':
            return a.createdAt?.localeCompare(b.createdAt!) ?? 0; // TODO proper date handling maybe
        }
      });
    return {app, newerVersions};
  }
}
