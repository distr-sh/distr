UPDATE Organization
SET features = array_append(features, 'vulnerabilities')
WHERE subscription_type IN ('business', 'enterprise')
  AND NOT ('vulnerabilities' = ANY (features));
