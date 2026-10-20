ALTER TABLE Organization RENAME COLUMN pre_connect_script TO docker_pre_connect_script;
ALTER TABLE Organization RENAME COLUMN post_connect_script TO docker_post_connect_script;
ALTER TABLE Organization RENAME COLUMN connect_script_is_sudo TO docker_connect_script_is_sudo;

ALTER TABLE Organization
  ADD COLUMN kubernetes_pre_connect_script TEXT,
  ADD COLUMN kubernetes_post_connect_script TEXT;
