export interface ContextSwitchPortal {
  id: string;
  name: string;
  imageUrl?: string;
}

/** The logo of an organization and the customer portals the user may switch to there, left out where it has neither. */
export interface ContextSwitchOrganization {
  id: string;
  logoUrl?: string;
  customerOrganizations: ContextSwitchPortal[];
}

export interface ContextSwitchTargets {
  organizations: ContextSwitchOrganization[];
}
