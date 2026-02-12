import {
  Application,
  ApplicationVersion,
  ApplicationVersionResource,
  DeploymentRequest,
  DeploymentTarget,
  DeploymentTargetAccessResponse,
} from '../types';
import {ConditionalPartial, defaultClientConfig} from './config';

export type ClientConfig = {
  /** The base URL of the Distr API ending with /api/v1, e.g. https://app.distr.sh/api/v1. */
  apiBase: string;
  /** The API key to authenticate with the Distr API. */
  apiKey: string;
};

export type ApplicationVersionFiles = {
  composeFile?: string;
  baseValuesFile?: string;
  templateFile?: string;
  resources?: ApplicationVersionResource[];
};

/**
 * The low-level Distr API client. Each method represents on API endpoint.
 */
export class Client {
  private readonly config: ClientConfig;

  constructor(config: ConditionalPartial<ClientConfig, keyof typeof defaultClientConfig>) {
    this.config = {
      apiKey: config.apiKey,
      apiBase: config.apiBase || defaultClientConfig.apiBase,
    };
    if (!this.config.apiBase.endsWith('/')) {
      this.config.apiBase += '/';
    }
  }

  public async getApplications(): Promise<Application[]> {
    return this.get<Application[]>('applications');
  }

  public async getApplication(applicationId: string): Promise<Application> {
    return this.get<Application>(`applications/${applicationId}`);
  }

  public async createApplication(application: Application): Promise<Application> {
    return this.post<Application>('applications', application);
  }

  public async updateApplication(application: Application): Promise<Application> {
    return this.put<Application>(`applications/${application.id}`, application);
  }

  public async createApplicationVersion(
    applicationId: string,
    version: ApplicationVersion,
    files?: ApplicationVersionFiles
  ): Promise<ApplicationVersion> {
    const formData = new FormData();
    formData.append('applicationversion', JSON.stringify(version));
    if (files?.composeFile) {
      formData.append('composefile', new Blob([files.composeFile], {type: 'application/yaml'}));
    }
    if (files?.baseValuesFile) {
      formData.append('valuesfile', new Blob([files.baseValuesFile], {type: 'application/yaml'}));
    }
    if (files?.templateFile) {
      formData.append('templatefile', new Blob([files.templateFile], {type: 'application/yaml'}));
    }
    if (files?.resources && files.resources.length > 0) {
      formData.append('resources', JSON.stringify(files.resources));
    }
    const path = `applications/${applicationId}/versions`;
    const response = await fetch(`${this.config.apiBase}${path}`, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        Authorization: `AccessToken ${this.config.apiKey}`,
      },
      body: formData,
    });
    return this.handleResponse<ApplicationVersion>(response, 'POST', path);
  }

  public async getApplicationVersionResources(
    applicationId: string,
    versionId: string
  ): Promise<ApplicationVersionResource[]> {
    return this.get<ApplicationVersionResource[]>(`applications/${applicationId}/versions/${versionId}/resources`);
  }

  public async getDeploymentTargets(): Promise<DeploymentTarget[]> {
    return this.get<DeploymentTarget[]>('deployment-targets');
  }

  public async getDeploymentTarget(deploymentTargetId: string): Promise<DeploymentTarget> {
    return this.get<DeploymentTarget>(`deployment-targets/${deploymentTargetId}`);
  }

  public async createDeploymentTarget(deploymentTarget: DeploymentTarget): Promise<DeploymentTarget> {
    return this.post<DeploymentTarget>('deployment-targets', deploymentTarget);
  }

  public async createOrUpdateDeployment(deploymentRequest: DeploymentRequest): Promise<DeploymentRequest> {
    return this.put<DeploymentRequest>('deployments', deploymentRequest);
  }

  public async createAccessForDeploymentTarget(deploymentTargetId: string): Promise<DeploymentTargetAccessResponse> {
    return this.post<DeploymentTargetAccessResponse>(`deployment-targets/${deploymentTargetId}/access-request`);
  }

  private async get<T>(path: string): Promise<T> {
    const response = await fetch(`${this.config.apiBase}${path}`, {
      method: 'GET',
      headers: {
        Accept: 'application/json',
        Authorization: `AccessToken ${this.config.apiKey}`,
      },
    });
    return await this.handleResponse<T>(response, 'GET', path);
  }

  private async post<T>(path: string, body?: T): Promise<T> {
    const response = await fetch(`${this.config.apiBase}${path}`, {
      method: 'POST',
      headers: {
        Accept: 'application/json',
        Authorization: `AccessToken ${this.config.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });
    return await this.handleResponse<T>(response, 'POST', path);
  }

  private async put<T>(path: string, body: T): Promise<T> {
    const response = await fetch(`${this.config.apiBase}${path}`, {
      method: 'PUT',
      headers: {
        Accept: 'application/json',
        Authorization: `AccessToken ${this.config.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });
    return await this.handleResponse<T>(response, 'PUT', path);
  }

  private async handleResponse<T>(response: Response, method: string, path: string) {
    if (response.status < 200 || response.status >= 300) {
      throw new Error(`${method} ${path} failed: ${response.status} ${response.statusText} "${await response.text()}"`);
    }
    const contentLength = response.headers.get('content-length');
    if (response.status === 204 || contentLength === '0') {
      return {} as T;
    }
    const text = await response.text();
    if (!text) {
      return {} as T;
    }
    return JSON.parse(text) as T;
  }
}
