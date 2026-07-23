UPDATE Organization
SET features = array_remove(features, 'custom_domains')
WHERE subscription_type = 'business';
