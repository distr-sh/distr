ALTER TYPE support_bundle_status RENAME TO support_bundle_status_old;
CREATE TYPE support_bundle_status AS ENUM ('initialized', 'created', 'resolved');

ALTER TABLE SupportBundle ADD COLUMN status_new support_bundle_status NOT NULL DEFAULT 'initialized';
UPDATE SupportBundle sb
SET status_new = CASE
  WHEN sb.status = 'resolved' THEN 'resolved'
  WHEN EXISTS (SELECT 1 FROM SupportBundleResource r WHERE r.support_bundle_id = sb.id) THEN 'created'
  ELSE 'initialized'
END::support_bundle_status;
ALTER TABLE SupportBundle DROP COLUMN status;
ALTER TABLE SupportBundle RENAME COLUMN status_new TO status;

DROP TYPE support_bundle_status_old;
