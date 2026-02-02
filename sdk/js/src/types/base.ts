export interface BaseModel {
  id?: string;
  createdAt?: string;
}

export interface Named {
  name?: string;
}

export interface TokenResponse {
  token: string;
}

export type LoginResponse = ({requiresMfa: false} & TokenResponse) | {requiresMfa: true};

export interface DeploymentTargetAccessResponse {
  connectUrl: string;
  targetId: string;
  targetSecret: string;
  connectCommand: string;
}
