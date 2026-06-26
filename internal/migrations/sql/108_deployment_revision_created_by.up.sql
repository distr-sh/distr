ALTER TABLE DeploymentRevision
  ADD COLUMN created_by_user_account_id UUID REFERENCES UserAccount (id) ON DELETE SET NULL;
