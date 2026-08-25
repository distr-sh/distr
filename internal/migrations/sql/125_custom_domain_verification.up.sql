-- A domain is only used for links, mails, agent manifests and registry URLs once it has been
-- verified. verified_at is the last successful check and is never cleared, verification_checked_at
-- is the last completed attempt whatever its outcome, and verification_error holds the user-facing
-- detail of the last definitive failure (an unresolvable lookup says nothing about the record and
-- therefore leaves both other columns alone).
ALTER TABLE CustomDomain
  ADD COLUMN verified_at             TIMESTAMP,
  ADD COLUMN verification_checked_at TIMESTAMP,
  ADD COLUMN verification_error      TEXT,
  ADD CONSTRAINT CustomDomain_verification_error_check
    CHECK (verification_error IS NULL OR verification_checked_at IS NOT NULL);
