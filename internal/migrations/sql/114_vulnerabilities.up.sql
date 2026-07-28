CREATE TYPE vulnerability_status AS ENUM ('triage', 'draft', 'published', 'resolved', 'canceled');

CREATE TYPE vulnerability_severity AS ENUM ('none', 'low', 'medium', 'high', 'critical');

CREATE TYPE vulnerability_version_relation AS ENUM ('affected', 'fixed');

CREATE TYPE vulnerability_event_type AS ENUM (
    'created',
    'status_changed',
    'edited',
    'tags_changed',
    'versions_changed',
    'reference_added',
    'reference_removed',
    'comment'
);

CREATE TABLE Vulnerability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    created_by_user_account_id UUID REFERENCES UserAccount (id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status vulnerability_status NOT NULL DEFAULT 'triage',
    severity vulnerability_severity NOT NULL DEFAULT 'none',
    cve_id TEXT,
    published_at TIMESTAMP,
    resolved_at TIMESTAMP
);

CREATE INDEX idx_vulnerability_organization_id ON Vulnerability (organization_id);

CREATE TABLE VulnerabilityReference (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vulnerability_id UUID NOT NULL REFERENCES Vulnerability (id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    label TEXT
);

CREATE INDEX idx_vulnerability_reference_vulnerability_id
    ON VulnerabilityReference (vulnerability_id);

CREATE TABLE VulnerabilityTag (
    vulnerability_id UUID NOT NULL REFERENCES Vulnerability (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    PRIMARY KEY (vulnerability_id, name)
);

CREATE TABLE VulnerabilityApplicationVersion (
    vulnerability_id UUID NOT NULL REFERENCES Vulnerability (id) ON DELETE CASCADE,
    application_version_id UUID NOT NULL REFERENCES ApplicationVersion (id) ON DELETE CASCADE,
    relation vulnerability_version_relation NOT NULL,
    PRIMARY KEY (vulnerability_id, application_version_id)
);

CREATE INDEX idx_vulnerability_application_version_application_version_id
    ON VulnerabilityApplicationVersion (application_version_id);

CREATE TABLE VulnerabilityArtifactVersion (
    vulnerability_id UUID NOT NULL REFERENCES Vulnerability (id) ON DELETE CASCADE,
    artifact_version_id UUID NOT NULL REFERENCES ArtifactVersion (id) ON DELETE CASCADE,
    relation vulnerability_version_relation NOT NULL,
    PRIMARY KEY (vulnerability_id, artifact_version_id)
);

CREATE INDEX idx_vulnerability_artifact_version_artifact_version_id
    ON VulnerabilityArtifactVersion (artifact_version_id);

CREATE TABLE VulnerabilityEvent (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    vulnerability_id UUID NOT NULL REFERENCES Vulnerability (id) ON DELETE CASCADE,
    user_account_id UUID REFERENCES UserAccount (id) ON DELETE SET NULL,
    type vulnerability_event_type NOT NULL,
    message TEXT
);

CREATE INDEX idx_vulnerability_event_vulnerability_id
    ON VulnerabilityEvent (vulnerability_id, created_at);

-- Must be the last statement: a newly added enum value cannot be used in the transaction
-- that adds it. No organization is granted the feature here for the same reason; the
-- Stripe webhook and the enterprise startup reconciliation both grant it from
-- FeaturesForSubscriptionType.
ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'vulnerabilities';
