ALTER TYPE support_bundle_status RENAME TO support_bundle_status_old;
CREATE TYPE support_bundle_status AS ENUM ('initialized', 'created', 'resolved');

ALTER TABLE SupportBundle ALTER COLUMN status DROP DEFAULT;
ALTER TABLE SupportBundle
  ALTER COLUMN status TYPE support_bundle_status
    USING (CASE
      WHEN status = 'resolved' THEN 'resolved'
      WHEN bundle_secret_expires_at IS NULL THEN 'created'
      ELSE 'initialized'
    END)::support_bundle_status;
ALTER TABLE SupportBundle ALTER COLUMN status SET DEFAULT 'initialized';

DROP TYPE support_bundle_status_old;
