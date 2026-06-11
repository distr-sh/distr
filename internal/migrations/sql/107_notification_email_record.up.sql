CREATE TABLE NotificationEmailRecord (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  email TEXT NOT NULL
);

CREATE INDEX idx_notification_email_record_email_created
  ON NotificationEmailRecord (email, created_at);
