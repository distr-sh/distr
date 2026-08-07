export interface CustomEmailConfiguration {
  id: string;
  createdAt: string;
  updatedAt: string;
  organizationId: string;
  enabled: boolean;
  fromAddress: string;
  smtpHost: string;
  smtpPort: number;
  smtpUsername: string;
  smtpPasswordSet: boolean;
  smtpImplicitTls: boolean;
}

export interface CustomEmailSettings {
  fromAddress: string;
  smtpHost: string;
  smtpPort: number;
  smtpUsername: string;
  /** Omitted to keep the stored password, empty to clear it. */
  smtpPassword?: string;
  smtpImplicitTls: boolean;
}

export interface UpdateCustomEmailConfigurationRequest extends CustomEmailSettings {
  enabled: boolean;
}
