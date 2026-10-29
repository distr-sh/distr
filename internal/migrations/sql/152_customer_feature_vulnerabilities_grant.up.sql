-- has to be a separate migration because of :
-- ERROR: unsafe use of new value "vulnerabilities" of enum type customer_organization_feature (SQLSTATE 55P04)

UPDATE CustomerOrganization co
  SET features = array_append(co.features, 'vulnerabilities')
  FROM Organization o
  WHERE o.id = co.organization_id
    AND 'vulnerabilities' = ANY(o.features)
    AND NOT 'vulnerabilities' = ANY(co.features);
