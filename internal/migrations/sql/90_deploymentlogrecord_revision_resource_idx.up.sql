CREATE INDEX deploymentlogrecord_deployment_revision_id_resource
    ON deploymentlogrecord (deployment_revision_id, resource);
