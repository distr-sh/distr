import {
  Advisory,
  AdvisoryDetail,
  AdvisoryEvent,
  AdvisoryFilter,
  AdvisoryImpact,
  Application,
  ApplicationVersion,
  ApplicationVersionResource,
  CreateAdvisoryCommentRequest,
  CreateAdvisoryRequest,
  CreateUpdateAdvisoryRequest,
  DeploymentRequest,
  DeploymentTarget,
  DeploymentTargetAccessResponse,
  UpdateAdvisoryStatusRequest,
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

  public async getAdvisories(filter: AdvisoryFilter = {}): Promise<Advisory[]> {
    const params = new URLSearchParams();
    for (const status of filter.status ?? []) {
      params.append('status', status);
    }
    for (const severity of filter.severity ?? []) {
      params.append('severity', severity);
    }
    for (const tag of filter.tag ?? []) {
      params.append('tag', tag);
    }
    const query = params.toString();
    return this.get<Advisory[]>(`advisories${query ? `?${query}` : ''}`);
  }

  public async getAdvisory(advisoryId: string): Promise<AdvisoryDetail> {
    return this.get<AdvisoryDetail>(`advisories/${advisoryId}`);
  }

  /** Spans undisclosed advisories and is therefore available to the vendor organization only. */
  public async getAdvisoryTags(): Promise<string[]> {
    return this.get<string[]>('advisories/tags');
  }

  public async getAdvisoryImpact(advisoryId: string): Promise<AdvisoryImpact> {
    return this.get<AdvisoryImpact>(`advisories/${advisoryId}/impact`);
  }

  public async createAdvisory(request: CreateAdvisoryRequest): Promise<AdvisoryDetail> {
    return this.post<AdvisoryDetail, CreateAdvisoryRequest>('advisories', request);
  }

  public async updateAdvisory(advisoryId: string, request: CreateUpdateAdvisoryRequest): Promise<AdvisoryDetail> {
    return this.put<AdvisoryDetail, CreateUpdateAdvisoryRequest>(`advisories/${advisoryId}`, request);
  }

  public async updateAdvisoryStatus(advisoryId: string, request: UpdateAdvisoryStatusRequest): Promise<AdvisoryDetail> {
    return this.patch<AdvisoryDetail, UpdateAdvisoryStatusRequest>(`advisories/${advisoryId}/status`, request);
  }

  public async createAdvisoryComment(
    advisoryId: string,
    request: CreateAdvisoryCommentRequest
  ): Promise<AdvisoryEvent> {
    return this.post<AdvisoryEvent, CreateAdvisoryCommentRequest>(`advisories/${advisoryId}/comments`, request);
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

  /** TBody defaults to TResponse so that the many endpoints echoing back their input stay concise. */
  private async post<TResponse, TBody = TResponse>(path: string, body?: TBody): Promise<TResponse> {
    return this.send<TResponse, TBody>('POST', path, body);
  }

  private async put<TResponse, TBody = TResponse>(path: string, body: TBody): Promise<TResponse> {
    return this.send<TResponse, TBody>('PUT', path, body);
  }

  private async patch<TResponse, TBody = TResponse>(path: string, body: TBody): Promise<TResponse> {
    return this.send<TResponse, TBody>('PATCH', path, body);
  }

  private async send<TResponse, TBody>(method: string, path: string, body?: TBody): Promise<TResponse> {
    const response = await fetch(`${this.config.apiBase}${path}`, {
      method,
      headers: {
        Accept: 'application/json',
        Authorization: `AccessToken ${this.config.apiKey}`,
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });
    return await this.handleResponse<TResponse>(response, method, path);
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
