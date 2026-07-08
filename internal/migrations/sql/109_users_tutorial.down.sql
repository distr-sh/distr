-- PostgreSQL does not support removing values from an enum type directly.
-- We need to recreate the type without the 'users' value.

DELETE FROM UserAccount_TutorialProgress WHERE tutorial = 'users';

CREATE TYPE TUTORIAL_NEW AS ENUM ('branding', 'agents', 'registry');

ALTER TABLE UserAccount_TutorialProgress
  ALTER COLUMN tutorial TYPE TUTORIAL_NEW
  USING tutorial::text::TUTORIAL_NEW;

DROP TYPE TUTORIAL;
ALTER TYPE TUTORIAL_NEW RENAME TO TUTORIAL;
