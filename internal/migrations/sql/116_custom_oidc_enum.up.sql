-- The new enum values must be added in their own migration: Postgres refuses to use a value
-- in an expression (the CHECK constraint of migration 117) in the same transaction that added it.
ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'custom_oidc_providers';

-- Identities provided by an organization's own OIDC configuration. 'generic' stays reserved for
-- the single env-configured generic instance provider behind /api/v1/auth/oidc/generic.
ALTER TYPE OIDC_PROVIDER ADD VALUE IF NOT EXISTS 'custom';
