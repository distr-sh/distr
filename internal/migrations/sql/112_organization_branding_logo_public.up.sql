-- The portal logo is now served without authentication via the public file API (the login page of a custom app
-- domain loads it as a plain resource), so existing logo files must be marked public.
UPDATE File
SET public = TRUE
FROM OrganizationBranding b
WHERE b.logo_image_id = File.id
  AND File.public = FALSE;
