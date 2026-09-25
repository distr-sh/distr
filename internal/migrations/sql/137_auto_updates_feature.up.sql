-- Separate from the migration adding the enum value, because a newly added value cannot be used in
-- the transaction that adds it.
UPDATE Organization SET features = array_append(features, 'auto_updates')
WHERE subscription_type IN ('business', 'enterprise') AND NOT 'auto_updates' = ANY(features);
