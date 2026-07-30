UPDATE Organization
  SET features = array_remove(features, 'vulnerabilities')
  WHERE 'vulnerabilities' = ANY (features);
