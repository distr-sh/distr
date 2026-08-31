CREATE TYPE advisory_status AS ENUM ('triage', 'draft', 'published', 'resolved', 'canceled');

CREATE TYPE advisory_severity AS ENUM ('none', 'low', 'medium', 'high', 'critical');

CREATE TYPE advisory_version_relation AS ENUM ('affected', 'fixed');

CREATE TYPE advisory_event_type AS ENUM ('published', 'status_changed', 'edited', 'comment');

CREATE TABLE Advisory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    updated_at TIMESTAMP NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    created_by_user_account_id UUID REFERENCES UserAccount (id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status advisory_status NOT NULL DEFAULT 'triage',
    severity advisory_severity NOT NULL DEFAULT 'none',
    cve_id TEXT,
    published_at TIMESTAMP,
    resolved_at TIMESTAMP
);

CREATE INDEX idx_advisory_organization_id ON Advisory (organization_id);

-- Deleting a user account nulls the column, which without an index scans the whole table.
CREATE INDEX idx_advisory_created_by_user_account_id
    ON Advisory (created_by_user_account_id);

-- One CVE is one issue, so an organization discloses it in exactly one advisory. Matching is
-- case-insensitive, otherwise "cve-2026-1234" would slip past as a second advisory for the
-- same CVE. Advisories without a CVE ID are unconstrained; there can be any number of them.
CREATE UNIQUE INDEX idx_advisory_organization_id_cve_id
    ON Advisory (organization_id, upper(cve_id))
    WHERE cve_id IS NOT NULL;

CREATE TABLE AdvisoryReference (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    advisory_id UUID NOT NULL REFERENCES Advisory (id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    label TEXT
);

CREATE INDEX idx_advisory_reference_advisory_id
    ON AdvisoryReference (advisory_id);

CREATE TABLE AdvisoryTag (
    advisory_id UUID NOT NULL REFERENCES Advisory (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    PRIMARY KEY (advisory_id, name)
);

CREATE TABLE AdvisoryApplicationVersion (
    advisory_id UUID NOT NULL REFERENCES Advisory (id) ON DELETE CASCADE,
    application_version_id UUID NOT NULL REFERENCES ApplicationVersion (id) ON DELETE CASCADE,
    relation advisory_version_relation NOT NULL,
    PRIMARY KEY (advisory_id, application_version_id)
);

CREATE INDEX idx_advisory_application_version_application_version_id
    ON AdvisoryApplicationVersion (application_version_id);

-- Cascading on artifact_version_id means a deleted or replaced tag drops the marking with
-- it. That is intended: registry tags are mutable and must stay deletable. The advisory
-- keeps its remaining markings, and one left with no affected version at all simply stops
-- being customer visible rather than becoming visible to everyone.
CREATE TABLE AdvisoryArtifactVersion (
    advisory_id UUID NOT NULL REFERENCES Advisory (id) ON DELETE CASCADE,
    artifact_version_id UUID NOT NULL REFERENCES ArtifactVersion (id) ON DELETE CASCADE,
    relation advisory_version_relation NOT NULL,
    PRIMARY KEY (advisory_id, artifact_version_id)
);

CREATE INDEX idx_advisory_artifact_version_artifact_version_id
    ON AdvisoryArtifactVersion (artifact_version_id);

CREATE TABLE AdvisoryEvent (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    advisory_id UUID NOT NULL REFERENCES Advisory (id) ON DELETE CASCADE,
    user_account_id UUID REFERENCES UserAccount (id) ON DELETE SET NULL,
    type advisory_event_type NOT NULL,
    message TEXT
);

CREATE INDEX idx_advisory_event_advisory_id
    ON AdvisoryEvent (advisory_id, created_at);

CREATE INDEX idx_advisory_event_user_account_id
    ON AdvisoryEvent (user_account_id);

-- The feature gate stays named after the "Vulnerability Management" category, which covers
-- security advisories today and will cover vulnerability scan logs later.
-- Must be the last statement: a newly added enum value cannot be used in the transaction
-- that adds it. No organization is granted the feature here for the same reason; the
-- Stripe webhook and the enterprise startup reconciliation both grant it from
-- FeaturesForSubscriptionType.
ALTER TYPE FEATURE ADD VALUE IF NOT EXISTS 'vulnerabilities';
