export interface CustomerLens {
  id: string;
  name: string;
}

/** The customer lenses the user may enter in an organization, left out where there are none. */
export interface OrganizationLenses {
  id: string;
  customerOrganizations: CustomerLens[];
}

export interface LensesResponse {
  organizations: OrganizationLenses[];
}
