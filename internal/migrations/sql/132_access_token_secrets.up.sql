ALTER TABLE AccessToken
  ADD COLUMN secret_1_salt BYTEA,
  ADD COLUMN secret_1_hash BYTEA,
  ADD COLUMN secret_1_created_at TIMESTAMP,
  ADD COLUMN secret_1_last_used_at TIMESTAMP,
  ADD COLUMN secret_2_salt BYTEA,
  ADD COLUMN secret_2_hash BYTEA,
  ADD COLUMN secret_2_created_at TIMESTAMP,
  ADD COLUMN secret_2_last_used_at TIMESTAMP,
  ADD CONSTRAINT AccessToken_secret_1_complete
    CHECK (num_nulls(secret_1_salt, secret_1_hash, secret_1_created_at) IN (0, 3)),
  ADD CONSTRAINT AccessToken_secret_2_complete
    CHECK (num_nulls(secret_2_salt, secret_2_hash, secret_2_created_at) IN (0, 3));
