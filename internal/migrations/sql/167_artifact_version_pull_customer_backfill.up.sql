UPDATE ArtifactVersionPull avpl
SET customer_organization_id = oua.customer_organization_id
FROM ArtifactVersion av
  JOIN Artifact a ON a.id = av.artifact_id
  JOIN Organization_UserAccount oua ON oua.organization_id = a.organization_id
WHERE av.id = avpl.artifact_version_id
  AND oua.user_account_id = avpl.useraccount_id
  AND avpl.customer_organization_id IS NULL
  AND oua.customer_organization_id IS NOT NULL;
