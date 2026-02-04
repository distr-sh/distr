-- has to be a separate migration because of :
-- ERROR: unsafe use of new value "notifications" of enum type customer_organization_feature (SQLSTATE 55P04)

UPDATE CustomerOrganization
  SET features = array_append(features, 'notifications')
  WHERE NOT 'notifications' = any(features);
