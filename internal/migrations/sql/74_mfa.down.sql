DROP TABLE UserAccount_MFARecoveryCode;

ALTER TABLE UserAccount
  DROP COLUMN mfa_secret,
  DROP COLUMN mfa_enabled,
  DROP COLUMN mfa_enabled_at;
