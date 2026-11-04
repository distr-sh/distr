ALTER TABLE CustomerOrganization DROP COLUMN controller_image_repository;
-- Enum values cannot be removed without recreating the type, which would fail for every value a
-- later migration added. The added values are harmless when unused.
