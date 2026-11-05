UPDATE File f
SET organization_id = co.organization_id
FROM CustomerOrganization co
WHERE co.image_id = f.id
  AND f.organization_id IS NULL
  AND NOT EXISTS (SELECT 1 FROM UserAccount u WHERE u.image_id = f.id);
