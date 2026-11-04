-- Separate from the migration adding the enum values, because a newly added value cannot be used in
-- the transaction that adds it.
UPDATE Organization SET features = array_append(features, 'customer_settings')
WHERE subscription_type IN ('business', 'enterprise') AND NOT 'customer_settings' = ANY(features);

UPDATE CustomerOrganization SET features = array_append(features, 'settings')
WHERE 'oidc_providers' = ANY(features) AND NOT 'settings' = ANY(features);
