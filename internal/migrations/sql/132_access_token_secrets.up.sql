ALTER TABLE AccessToken
  ADD COLUMN secret_1_hash BYTEA,
  ADD COLUMN secret_1_created_at TIMESTAMP,
  ADD COLUMN secret_1_expires_at TIMESTAMP,
  ADD COLUMN secret_1_last_used_at TIMESTAMP,
  ADD COLUMN secret_2_hash BYTEA,
  ADD COLUMN secret_2_created_at TIMESTAMP,
  ADD COLUMN secret_2_expires_at TIMESTAMP,
  ADD COLUMN secret_2_last_used_at TIMESTAMP,
  ADD CONSTRAINT AccessToken_secret_1_complete
    CHECK (num_nulls(secret_1_hash, secret_1_created_at) IN (0, 2)
      AND (secret_1_hash IS NOT NULL OR secret_1_expires_at IS NULL)),
  ADD CONSTRAINT AccessToken_secret_2_complete
    CHECK (num_nulls(secret_2_hash, secret_2_created_at) IN (0, 2)
      AND (secret_2_hash IS NOT NULL OR secret_2_expires_at IS NULL)),
  ADD CONSTRAINT AccessToken_legacy_expires_at
    CHECK (expires_at IS NULL OR num_nulls(secret_1_hash, secret_2_hash) = 2);

COMMENT ON COLUMN AccessToken.expires_at IS
  'The expiration of a token issued before secrets existed. A token with secrets expires with them.';
