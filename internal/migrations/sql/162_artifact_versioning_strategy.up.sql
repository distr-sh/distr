ALTER TABLE Artifact
  ADD COLUMN versioning_strategy VERSIONING_STRATEGY NOT NULL DEFAULT 'chronological';
