ALTER TABLE UserAccount
  ADD COLUMN is_super_admin BOOLEAN NOT NULL DEFAULT false;
