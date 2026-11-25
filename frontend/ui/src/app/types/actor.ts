/** Missing the name and image of a user the viewer may not know, who is then known by their organization. */
export interface Actor {
  name?: string;
  imageUrl?: string;
  customerOrganizationId?: string;
  partnerOrganizationId?: string;
  deleted?: boolean;
}
