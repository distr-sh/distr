/** The customer or partner organization a lens shows, exactly one of which is set. */
export interface Lens {
  customerOrganizationId?: string;
  partnerOrganizationId?: string;
}

export interface ContextSwitchPortal {
  id: string;
  name: string;
  imageUrl?: string;
}

/** The logo of an organization and the portals the user may switch to there, left out where it has neither. */
export interface ContextSwitchOrganization {
  id: string;
  logoUrl?: string;
  customerOrganizations: ContextSwitchPortal[];
  partnerOrganizations: ContextSwitchPortal[];
}

export interface ContextSwitchTargets {
  organizations: ContextSwitchOrganization[];
}
