CREATE TABLE ApplicationNotificationConfiguration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    customer_organization_id UUID REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    update_available_trigger_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    customer_message TEXT
);

CREATE INDEX idx_application_notification_configuration_organization_id
    ON ApplicationNotificationConfiguration (organization_id);

CREATE TABLE ApplicationNotificationConfiguration_Application (
    application_notification_configuration_id UUID NOT NULL
        REFERENCES ApplicationNotificationConfiguration (id) ON DELETE CASCADE,
    application_id UUID NOT NULL REFERENCES Application (id) ON DELETE CASCADE,
    PRIMARY KEY (application_notification_configuration_id, application_id)
);

CREATE INDEX idx_application_notification_configuration_application_id
    ON ApplicationNotificationConfiguration_Application (application_id);

CREATE TABLE ApplicationNotificationConfiguration_Organization_UserAccount (
    application_notification_configuration_id UUID NOT NULL
        REFERENCES ApplicationNotificationConfiguration (id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    user_account_id UUID NOT NULL REFERENCES UserAccount (id) ON DELETE CASCADE,
    PRIMARY KEY (application_notification_configuration_id, organization_id, user_account_id),
    FOREIGN KEY (organization_id, user_account_id)
        REFERENCES Organization_UserAccount (organization_id, user_account_id) ON DELETE CASCADE
);

CREATE TABLE ArtifactNotificationConfiguration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    customer_organization_id UUID REFERENCES CustomerOrganization (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    new_version_trigger_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    customer_message TEXT
);

CREATE INDEX idx_artifact_notification_configuration_organization_id
    ON ArtifactNotificationConfiguration (organization_id);

CREATE TABLE ArtifactNotificationConfiguration_Artifact (
    artifact_notification_configuration_id UUID NOT NULL
        REFERENCES ArtifactNotificationConfiguration (id) ON DELETE CASCADE,
    artifact_id UUID NOT NULL REFERENCES Artifact (id) ON DELETE CASCADE,
    PRIMARY KEY (artifact_notification_configuration_id, artifact_id)
);

CREATE INDEX idx_artifact_notification_configuration_artifact_id
    ON ArtifactNotificationConfiguration_Artifact (artifact_id);

CREATE TABLE ArtifactNotificationConfiguration_Organization_UserAccount (
    artifact_notification_configuration_id UUID NOT NULL
        REFERENCES ArtifactNotificationConfiguration (id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES Organization (id) ON DELETE CASCADE,
    user_account_id UUID NOT NULL REFERENCES UserAccount (id) ON DELETE CASCADE,
    PRIMARY KEY (artifact_notification_configuration_id, organization_id, user_account_id),
    FOREIGN KEY (organization_id, user_account_id)
        REFERENCES Organization_UserAccount (organization_id, user_account_id) ON DELETE CASCADE
);
