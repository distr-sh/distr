CREATE TYPE OIDC_FLOW AS ENUM ('login', 'registration');

-- The default is the flow that never creates an account, so a state row that does not set it
-- cannot register a user.
ALTER TABLE OIDCState ADD COLUMN flow OIDC_FLOW NOT NULL DEFAULT 'login';
