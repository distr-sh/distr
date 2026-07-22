-- Revert 'business' back to 'starter' in the SubscriptionType enum.
-- Business organizations are converted to pro first; starter organizations
-- converted to pro by the up migration cannot be restored.

UPDATE Organization
  SET subscription_type = 'pro'
  WHERE subscription_type = 'business';

ALTER TYPE SubscriptionType RENAME VALUE 'business' TO 'starter';
