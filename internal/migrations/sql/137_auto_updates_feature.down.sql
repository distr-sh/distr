UPDATE Organization SET features = array_remove(features, 'auto_updates')
WHERE 'auto_updates' = ANY(features);
