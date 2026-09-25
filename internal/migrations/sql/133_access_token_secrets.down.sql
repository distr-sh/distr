-- A token that has a secret is authenticated by it, and the key it is identified by is handed out
-- as its id. Keeping such a row across the rollback would turn that identifier back into the whole
-- credential for the binary that runs afterwards, so the rollback revokes those tokens instead.
DELETE FROM AccessToken
WHERE secret_1_hash IS NOT NULL OR secret_2_hash IS NOT NULL;

ALTER TABLE AccessToken
  DROP CONSTRAINT AccessToken_secret_1_complete,
  DROP CONSTRAINT AccessToken_secret_2_complete,
  DROP CONSTRAINT AccessToken_legacy_expires_at,
  DROP COLUMN secret_1_hash,
  DROP COLUMN secret_1_created_at,
  DROP COLUMN secret_1_expires_at,
  DROP COLUMN secret_1_last_used_at,
  DROP COLUMN secret_2_hash,
  DROP COLUMN secret_2_created_at,
  DROP COLUMN secret_2_expires_at,
  DROP COLUMN secret_2_last_used_at;

COMMENT ON COLUMN AccessToken.expires_at IS NULL;
