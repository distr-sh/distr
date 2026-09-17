CREATE INDEX fk_UserAccount_image_id ON UserAccount (image_id) WHERE image_id IS NOT NULL;
CREATE INDEX fk_Application_image_id ON Application (image_id) WHERE image_id IS NOT NULL;
CREATE INDEX fk_Artifact_image_id ON Artifact (image_id) WHERE image_id IS NOT NULL;
CREATE INDEX fk_CustomerOrganization_image_id ON CustomerOrganization (image_id)
  WHERE image_id IS NOT NULL;
CREATE INDEX fk_OrganizationBranding_logo_image_id ON OrganizationBranding (logo_image_id)
  WHERE logo_image_id IS NOT NULL;
CREATE INDEX fk_OrganizationBranding_favicon_image_id ON OrganizationBranding (favicon_image_id)
  WHERE favicon_image_id IS NOT NULL;
