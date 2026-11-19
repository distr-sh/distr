UPDATE Organization SET features = array_remove(features, 'support')
WHERE 'support' = ANY(features);

UPDATE Organization SET features = array_remove(features, 'notifications')
WHERE 'notifications' = ANY(features);

UPDATE Organization SET features = array_remove(features, 'user_roles')
WHERE 'user_roles' = ANY(features);
