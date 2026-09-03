UPDATE File
SET public = FALSE
FROM OrganizationBranding b
WHERE b.logo_image_id = File.id;
