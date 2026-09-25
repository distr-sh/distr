ALTER TABLE SupportBundle
  ALTER COLUMN created_by_user_account_id DROP NOT NULL,
  DROP CONSTRAINT supportbundle_created_by_user_account_id_fkey,
  ADD CONSTRAINT supportbundle_created_by_user_account_id_fkey
    FOREIGN KEY (created_by_user_account_id) REFERENCES UserAccount (id) ON DELETE SET NULL,
  DROP CONSTRAINT supportbundle_status_changed_by_user_account_id_fkey,
  ADD CONSTRAINT supportbundle_status_changed_by_user_account_id_fkey
    FOREIGN KEY (status_changed_by_user_account_id) REFERENCES UserAccount (id) ON DELETE SET NULL;

ALTER TABLE SupportBundleComment
  ALTER COLUMN user_account_id DROP NOT NULL,
  DROP CONSTRAINT supportbundlecomment_user_account_id_fkey,
  ADD CONSTRAINT supportbundlecomment_user_account_id_fkey
    FOREIGN KEY (user_account_id) REFERENCES UserAccount (id) ON DELETE SET NULL;

CREATE INDEX fk_UserAccount_image_id ON UserAccount (image_id) WHERE image_id IS NOT NULL;
CREATE INDEX fk_Application_image_id ON Application (image_id) WHERE image_id IS NOT NULL;
CREATE INDEX fk_Artifact_image_id ON Artifact (image_id) WHERE image_id IS NOT NULL;
CREATE INDEX fk_CustomerOrganization_image_id ON CustomerOrganization (image_id)
  WHERE image_id IS NOT NULL;
CREATE INDEX fk_OrganizationBranding_logo_image_id ON OrganizationBranding (logo_image_id)
  WHERE logo_image_id IS NOT NULL;
CREATE INDEX fk_OrganizationBranding_favicon_image_id ON OrganizationBranding (favicon_image_id)
  WHERE favicon_image_id IS NOT NULL;
