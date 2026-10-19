ALTER TABLE ApplicationEntitlement
  ADD COLUMN all_versions BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN auto_add_version_regex TEXT,
  ADD CONSTRAINT applicationentitlement_all_versions_no_regex
    CHECK (NOT (all_versions AND auto_add_version_regex IS NOT NULL));

UPDATE ApplicationEntitlement ae SET all_versions = TRUE
WHERE NOT EXISTS (
  SELECT FROM ApplicationEntitlement_ApplicationVersion aeav WHERE aeav.application_entitlement_id = ae.id
);

ALTER TABLE ApplicationEntitlement ALTER COLUMN all_versions DROP DEFAULT;
