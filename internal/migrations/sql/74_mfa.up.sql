ALTER TABLE UserAccount
  ADD COLUMN mfa_secret TEXT,
  ADD COLUMN mfa_enabled BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN mfa_enabled_at TIMESTAMP WITH TIME ZONE,
  ADD CONSTRAINT mfa_secret_not_null_if_enabled
    CHECK (mfa_enabled = false OR mfa_secret IS NOT NULL);

CREATE TABLE UserAccount_MFARecoveryCode (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP DEFAULT current_timestamp,
  user_account_id UUID NOT NULL REFERENCES UserAccount(id) ON DELETE CASCADE,
  code_hash BYTEA NOT NULL,
  code_salt BYTEA NOT NULL,
  used_at TIMESTAMP
);

CREATE INDEX idx_UserAccount_MFARecoveryCode_used_at
  ON UserAccount_MFARecoveryCode(user_account_id, used_at);
