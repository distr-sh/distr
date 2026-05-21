import {AccountRole} from './user-account';

export interface ServiceAccount {
  id: string;
  createdAt: string;
  name: string;
  accountRole: AccountRole;
  customerOrganizationId?: string;
}

export interface CreateServiceAccountRequest {
  name: string;
  accountRole: AccountRole;
  customerOrganizationId?: string;
}

export interface PatchServiceAccountRequest {
  name?: string;
  accountRole?: AccountRole;
}
