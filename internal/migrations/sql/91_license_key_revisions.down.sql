ALTER TABLE LicenseKey
    ADD COLUMN not_before TIMESTAMP,
    ADD COLUMN expires_at TIMESTAMP,
    ADD COLUMN payload    JSONB DEFAULT '{}';

UPDATE LicenseKey lk
SET
    not_before = r.not_before,
    expires_at = r.expires_at,
    payload    = r.payload
FROM (
    SELECT DISTINCT ON (license_key_id)
        license_key_id, not_before, expires_at, payload
    FROM LicenseKeyRevision
    ORDER BY license_key_id, created_at DESC, id DESC
) r
WHERE lk.id = r.license_key_id;

ALTER TABLE LicenseKey
    ALTER COLUMN not_before SET NOT NULL,
    ALTER COLUMN expires_at SET NOT NULL,
    ALTER COLUMN payload    SET NOT NULL;

DROP TABLE LicenseKeyRevision;
