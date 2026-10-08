-- Preserve notices already saved during development under the new name.
ALTER TABLE recipes RENAME COLUMN notice TO reminder;
