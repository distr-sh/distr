UPDATE Organization
SET features = array_append(features, 'vulnerabilities')
WHERE subscription_type = 'business'
  AND NOT ('vulnerabilities' = ANY (features));
