-- Has to be a separate migration from 114: a newly added enum value cannot be used in the
-- same transaction that adds it (SQLSTATE 55P04). Grant the vulnerabilities feature to
-- existing Business and Enterprise organizations so paying tenants are not locked out of
-- Security Advisories until their next Stripe webhook or startup reconciliation fires.
UPDATE Organization
  SET features = array_append(features, 'vulnerabilities')
  WHERE subscription_type IN ('business', 'enterprise')
    AND NOT 'vulnerabilities' = ANY (features);
