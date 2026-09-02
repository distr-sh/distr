UPDATE Organization
  SET features = array_remove(features, 'vulnerabilities')
  WHERE subscription_type = 'business';
