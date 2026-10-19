ALTER TABLE ApplicationEntitlement
  DROP CONSTRAINT applicationentitlement_all_versions_no_regex,
  DROP COLUMN all_versions,
  DROP COLUMN auto_add_version_regex;
