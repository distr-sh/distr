CREATE TABLE LicenseKeyRevision (
    id             UUID      PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at     TIMESTAMP NOT NULL DEFAULT current_timestamp,
    license_key_id UUID      NOT NULL REFERENCES LicenseKey(id) ON DELETE CASCADE,
    not_before     TIMESTAMP NOT NULL,
    expires_at     TIMESTAMP NOT NULL,
    payload        JSONB     NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_licensekeyrevision_license_key_id_created_at_desc
    ON LicenseKeyRevision (license_key_id, created_at DESC);

INSERT INTO LicenseKeyRevision (created_at, license_key_id, not_before, expires_at, payload)
SELECT lk.created_at, lk.id, lk.not_before, lk.expires_at, lk.payload
FROM LicenseKey lk;

ALTER TABLE LicenseKey
    DROP COLUMN not_before,
    DROP COLUMN expires_at,
    DROP COLUMN payload;
