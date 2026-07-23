-- Backfill the custom_domains feature for organizations already on the business plan.
-- Migration 114 only added the enum value; startup reconciliation never grants plan
-- features (only the Stripe webhook does), so existing business orgs would stay gated
-- until an unrelated subscription update runs. This runs in its own migration because
-- a newly added enum value cannot be used in the same transaction that added it.
UPDATE Organization
SET features = array_append(features, 'custom_domains')
WHERE subscription_type = 'business'
  AND NOT ('custom_domains' = ANY(features));
