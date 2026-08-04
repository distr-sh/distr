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

export type LoginResponse =
  // redirectUrl is set when the user's organization has a custom app domain and the login happened on the
  // instance's default host: a browser is expected to continue there. The token is valid on either host.
  ({requiresMfa: false; redirectUrl?: string} & TokenResponse) | {requiresMfa: true};

export interface DeploymentTargetAccessResponse {
  connectUrl: string;
  targetId: string;
  targetSecret: string;
  connectCommand: string;
}
