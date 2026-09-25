-- ALTER TYPE FEATURE DROP VALUE is not supported by postgres, and recreating the type would fail
-- for every value a later migration added that an organization still holds. The value is harmless
-- when unused.
UPDATE Organization SET features = array_remove(features, 'pre_post_scripts')
WHERE 'pre_post_scripts' = ANY(features);

ALTER TABLE Organization
  DROP COLUMN pre_connect_script,
  DROP COLUMN post_connect_script;
