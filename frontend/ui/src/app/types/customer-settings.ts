export interface CustomerSettings {
  controllerImageRepository?: string;
  defaultControllerImageRepository: string;
}

export interface UpdateCustomerSettingsRequest {
  controllerImageRepository?: string;
}
