import {BaseModel} from '@distr-sh/distr-sdk';

export interface DeploymentLogRecord extends BaseModel {
  deploymentId: string;
  deploymentRevisionId: string;
  resource: string;
  timestamp: string;
  severity: string;
  body: string;
}

export interface DeploymentLogRecordResources {
  active: string[];
  archived: string[];
}
