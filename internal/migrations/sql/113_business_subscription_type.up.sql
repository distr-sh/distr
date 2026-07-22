-- Replace 'starter' with 'business' in the SubscriptionType enum.
-- Existing starter organizations are converted to pro first, so the now-unused
-- 'starter' value can simply be renamed instead of recreating the type.

UPDATE Organization
  SET subscription_type = 'pro'
  WHERE subscription_type = 'starter';

ALTER TYPE SubscriptionType RENAME VALUE 'starter' TO 'business';
