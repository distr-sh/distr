CREATE TABLE UpdateNotificationConfiguration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    customer_organization_id UUID REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_update_notification_configuration_organization_id
    ON UpdateNotificationConfiguration (organization_id);

CREATE TABLE UpdateNotificationConfiguration_Application (
    update_notification_configuration_id UUID NOT NULL
        REFERENCES UpdateNotificationConfiguration (id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES Application (id) ON DELETE CASCADE,
    PRIMARY KEY (update_notification_configuration_id, application_id)
);

CREATE INDEX idx_update_notification_configuration_application_id
    ON UpdateNotificationConfiguration_Application (application_id);

CREATE TABLE UpdateNotificationConfiguration_Artifact (
    update_notification_configuration_id UUID NOT NULL
        REFERENCES UpdateNotificationConfiguration (id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES Artifact (id) ON DELETE CASCADE,
    PRIMARY KEY (update_notification_configuration_id, artifact_id)
);

CREATE INDEX idx_update_notification_configuration_artifact_id
    ON UpdateNotificationConfiguration_Artifact (artifact_id);

CREATE TABLE UpdateNotificationConfiguration_Organization_UserAccount (
    update_notification_configuration_id UUID NOT NULL
        REFERENCES UpdateNotificationConfiguration (id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    user_account_id UUID NOT NULL REFERENCES UserAccount (id) ON DELETE CASCADE,
    PRIMARY KEY (update_notification_configuration_id, organization_id, user_account_id),
    FOREIGN KEY (organization_id, user_account_id)
        REFERENCES Organization_UserAccount (organization_id, user_account_id) ON DELETE CASCADE
);

ALTER TYPE CUSTOMER_ORGANIZATION_FEATURE ADD VALUE IF NOT EXISTS 'update_notifications';

ALTER TYPE NOTIFICATION_RECORD_TYPE ADD VALUE IF NOT EXISTS 'update_available';

ALTER TABLE NotificationRecord
    ADD COLUMN update_notification_configuration_id UUID
        REFERENCES UpdateNotificationConfiguration (id) ON DELETE CASCADE,
    ADD COLUMN application_version_id UUID REFERENCES ApplicationVersion (id) ON DELETE CASCADE,
    ADD COLUMN artifact_version_id UUID REFERENCES ArtifactVersion (id) ON DELETE CASCADE,
    ADD COLUMN partner_organization_id UUID REFERENCES PartnerOrganization (id) ON DELETE CASCADE,
    ADD COLUMN deployments JSONB,
    ADD CONSTRAINT notification_record_update_subject CHECK (
        update_notification_configuration_id IS NULL
        OR (alert_configuration_id IS NULL AND num_nonnulls(application_version_id, artifact_version_id) = 1)
    );

CREATE INDEX fk_NotificationRecord_application_version_id
    ON NotificationRecord (application_version_id);
CREATE INDEX fk_NotificationRecord_artifact_version_id
    ON NotificationRecord (artifact_version_id);
CREATE INDEX fk_NotificationRecord_partner_organization_id
    ON NotificationRecord (partner_organization_id);

-- One record per configuration, version and audience, which is the vendor's own team, one partner
-- or one customer. Reserving it is what lets only one of two racing sends through, and a customer
-- an entitlement change covers later still finds its audience free.
CREATE UNIQUE INDEX idx_notification_record_update_audience
    ON NotificationRecord (
        update_notification_configuration_id,
        application_version_id,
        artifact_version_id,
        customer_organization_id,
        partner_organization_id
    ) NULLS NOT DISTINCT
    WHERE update_notification_configuration_id IS NOT NULL;
