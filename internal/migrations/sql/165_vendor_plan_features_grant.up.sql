-- Separate from the migration adding the enum values, because a newly added value cannot be used in
-- the transaction that adds it.
UPDATE Organization SET features = array_append(features, 'support')
WHERE subscription_type <> 'community' AND NOT 'support' = ANY(features);

UPDATE Organization SET features = array_append(features, 'notifications')
WHERE subscription_type <> 'community' AND NOT 'notifications' = ANY(features);

UPDATE Organization SET features = array_append(features, 'user_roles')
WHERE subscription_type <> 'community' AND NOT 'user_roles' = ANY(features);
