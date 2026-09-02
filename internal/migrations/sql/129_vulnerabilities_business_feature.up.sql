-- Has to be a separate migration because of:
-- ERROR: unsafe use of new value "vulnerabilities" of enum type feature (SQLSTATE 55P04)
--
-- Enterprise organizations are reconciled on startup, Business ones only once their Stripe
-- subscription next changes, which is why they are granted the feature here.
UPDATE Organization
  SET features = array_append(features, 'vulnerabilities')
  WHERE subscription_type = 'business' AND NOT 'vulnerabilities' = any(features);
