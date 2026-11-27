UPDATE SupportBundle SET status = 'resolved' WHERE status = 'canceled';

ALTER TYPE support_bundle_status RENAME TO support_bundle_status_old;
CREATE TYPE support_bundle_status AS ENUM ('initialized', 'created', 'resolved');

ALTER TABLE SupportBundle ALTER COLUMN status DROP DEFAULT;
ALTER TABLE SupportBundle
  ALTER COLUMN status TYPE support_bundle_status USING status::text::support_bundle_status;
ALTER TABLE SupportBundle ALTER COLUMN status SET DEFAULT 'initialized';

DROP TYPE support_bundle_status_old;
