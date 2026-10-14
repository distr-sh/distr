/** A customer or partner portal of an organization, the user's own portal when neither is set. */
export interface Portal {
  customerOrganizationId?: string;
  partnerOrganizationId?: string;
}

export interface SwitchContextRequest extends Portal {
  organizationId: string;
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
