ALTER TABLE CustomDomain
  DROP CONSTRAINT CustomDomain_verification_error_check,
  DROP COLUMN verification_error,
  DROP COLUMN verification_checked_at,
  DROP COLUMN verified_at;
