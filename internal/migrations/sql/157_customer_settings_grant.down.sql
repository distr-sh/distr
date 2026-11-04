UPDATE CustomerOrganization SET features = array_remove(features, 'settings')
WHERE 'settings' = ANY(features);

UPDATE Organization SET features = array_remove(features, 'customer_settings')
WHERE 'customer_settings' = ANY(features);
