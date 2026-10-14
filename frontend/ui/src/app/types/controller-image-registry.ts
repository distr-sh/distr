export interface ControllerImageRegistry {
  imageRegistry?: string;
  defaultImageRegistry: string;
}

export interface UpdateControllerImageRegistryRequest {
  imageRegistry: string;
  customerOrganizationId?: string;
}
