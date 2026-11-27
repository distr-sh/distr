UPDATE SupportBundle
SET status_changed_at = NULL, status_changed_by_user_account_id = NULL
WHERE status = 'created';

ALTER TYPE support_bundle_status RENAME TO support_bundle_status_old;
CREATE TYPE support_bundle_status AS ENUM ('open', 'resolved');

ALTER TABLE SupportBundle ALTER COLUMN status DROP DEFAULT;
ALTER TABLE SupportBundle
  ALTER COLUMN status TYPE support_bundle_status
    USING (CASE WHEN status = 'resolved' THEN 'resolved' ELSE 'open' END)::support_bundle_status;
ALTER TABLE SupportBundle ALTER COLUMN status SET DEFAULT 'open';

DROP TYPE support_bundle_status_old;
