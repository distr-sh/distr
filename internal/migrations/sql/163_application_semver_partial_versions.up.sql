-- The semver strategy accepted names with a missing minor or patch number until it began to reject
-- them. An application holding such a name could no longer add or rename a version, so it moves to
-- the chronological strategy, which imposes no requirement on the names.
UPDATE Application a
SET versioning_strategy = 'chronological'
WHERE a.versioning_strategy = 'semver'
  AND EXISTS (
    SELECT 1 FROM ApplicationVersion av
    WHERE av.application_id = a.id
      AND av.name !~ '^v?[0-9]+\.[0-9]+\.[0-9]+([-+]|$)'
  );
