ALTER TYPE TUTORIAL RENAME VALUE 'controllers' TO 'agents';

ALTER TABLE DeploymentTargetMetrics RENAME COLUMN controller_log_bytes TO agent_log_bytes;
ALTER TABLE DeploymentTargetMetrics RENAME COLUMN controller_memory_bytes TO agent_memory_bytes;
ALTER TABLE DeploymentTargetMetrics RENAME COLUMN controller_cpu_usage_millis TO agent_cpu_usage_millis;

ALTER TABLE DeploymentTarget DROP COLUMN legacy_agent_manifest;

ALTER INDEX fk_DeploymentTarget_controller_version_id RENAME TO fk_DeploymentTarget_agent_version_id;
ALTER TABLE DeploymentTarget
  RENAME CONSTRAINT deploymenttarget_reported_controller_version_id_fkey
    TO deploymenttarget_reported_agent_version_id_fkey;
ALTER TABLE DeploymentTarget
  RENAME CONSTRAINT deploymenttarget_controller_version_id_fkey TO deploymenttarget_agent_version_id_fkey;
ALTER TABLE DeploymentTarget RENAME COLUMN reported_controller_version_id TO reported_agent_version_id;
ALTER TABLE DeploymentTarget RENAME COLUMN controller_version_id TO agent_version_id;

ALTER INDEX ControllerVersion_name RENAME TO AgentVersion_name;
ALTER TABLE ControllerVersion RENAME CONSTRAINT controllerversion_name_key TO agentversion_name_key;
ALTER TABLE ControllerVersion RENAME CONSTRAINT controllerversion_pkey TO agentversion_pkey;
ALTER TABLE ControllerVersion RENAME TO AgentVersion;
