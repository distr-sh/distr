ALTER TABLE Organization
  ADD COLUMN kubernetes_pre_connect_script TEXT,
  ADD COLUMN kubernetes_post_connect_script TEXT;
