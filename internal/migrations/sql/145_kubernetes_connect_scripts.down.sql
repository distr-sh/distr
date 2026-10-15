ALTER TABLE Organization
  DROP COLUMN kubernetes_pre_connect_script,
  DROP COLUMN kubernetes_post_connect_script;

ALTER TABLE Organization RENAME COLUMN docker_pre_connect_script TO pre_connect_script;
ALTER TABLE Organization RENAME COLUMN docker_post_connect_script TO post_connect_script;
ALTER TABLE Organization RENAME COLUMN docker_connect_script_is_sudo TO connect_script_is_sudo;
