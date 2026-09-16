-- The binary that runs after the rollback accepts the key only for a row that has no secret at all,
-- so the key of a row that is mid-migration stops authenticating and the expiration it carried
-- describes nothing anymore.
UPDATE AccessToken
SET expires_at = NULL
WHERE secret_1_hash IS NOT NULL OR secret_2_hash IS NOT NULL;

ALTER TABLE AccessToken
  DROP CONSTRAINT AccessToken_key_credential,
  ADD CONSTRAINT AccessToken_legacy_expires_at
    CHECK (expires_at IS NULL OR num_nulls(secret_1_hash, secret_2_hash) = 2),
  DROP COLUMN key_is_credential,
  DROP COLUMN key_last_used_at;

COMMENT ON COLUMN AccessToken.expires_at IS
  'The expiration of a token issued before secrets existed. A token with secrets expires with them.';
