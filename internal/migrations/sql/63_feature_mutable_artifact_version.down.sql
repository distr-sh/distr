-- ALTER TYPE FEATURE DROP VALUE is not supported by postgres, and recreating the type would fail
-- for every value a later migration added that an organization still holds. The value is harmless
-- when unused.
UPDATE Organization SET features = array_remove(features, 'artifact_version_mutable')
WHERE 'artifact_version_mutable' = ANY(features);
