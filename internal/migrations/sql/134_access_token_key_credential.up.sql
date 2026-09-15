ALTER TABLE AccessToken
  ADD COLUMN key_is_credential BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN key_last_used_at TIMESTAMP;

-- A row without a secret is authenticated by its key alone, which is what the column records, and
-- every use it has seen was made with that key. One that was issued without an expiration gets 90
-- days from here, so that a credential kept in plain text stops working even if nobody migrates it.
UPDATE AccessToken
SET key_is_credential = true,
    key_last_used_at = last_used_at,
    expires_at = coalesce(expires_at, now() + INTERVAL '90 days')
WHERE secret_1_hash IS NULL AND secret_2_hash IS NULL;

ALTER TABLE AccessToken
  DROP CONSTRAINT AccessToken_legacy_expires_at,
  ADD CONSTRAINT AccessToken_key_credential
    CHECK ((key_is_credential OR num_nulls(expires_at, key_last_used_at) = 2)
      AND (NOT key_is_credential OR expires_at IS NOT NULL)
      AND num_nonnulls(secret_1_hash, secret_2_hash) + key_is_credential::INT BETWEEN 1 AND 2);

COMMENT ON COLUMN AccessToken.expires_at IS
  'The expiration of the key while the key itself is the credential. A secret expires on its own.';

COMMENT ON COLUMN AccessToken.key_is_credential IS
  'Whether the key authenticates on its own, which is the format that predates secrets.';
