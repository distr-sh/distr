UPDATE File f
SET organization_id = NULL
FROM CustomerOrganization co
WHERE co.image_id = f.id;
