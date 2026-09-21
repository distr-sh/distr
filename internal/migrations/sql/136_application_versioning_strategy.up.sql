ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'auto_updates';

CREATE TYPE VERSIONING_STRATEGY AS ENUM ('semver', 'chronological', 'legacy');

-- Existing applications become 'legacy' because their version names were never validated, so
-- neither 'semver' nor 'chronological' can be assumed to order them the way the vendor expects.
ALTER TABLE Application
  ADD COLUMN versioning_strategy VERSIONING_STRATEGY NOT NULL DEFAULT 'legacy',
  ADD COLUMN allow_automatic_updates BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE Deployment
  ADD COLUMN automatic_application_updates_enabled BOOLEAN NOT NULL DEFAULT false;

CREATE TYPE DEPLOYMENT_REVISION_TRIGGER AS ENUM (
  'user',
  'automatic_update',
  'secret_change',
  'license_key_change'
);

ALTER TABLE DeploymentRevision
  ADD COLUMN trigger DEPLOYMENT_REVISION_TRIGGER NOT NULL DEFAULT 'user';
