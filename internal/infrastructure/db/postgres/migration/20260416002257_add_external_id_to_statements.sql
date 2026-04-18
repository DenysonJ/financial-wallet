-- +goose Up
ALTER TABLE statements ADD COLUMN external_id VARCHAR(255);
ALTER TABLE statements ADD COLUMN posted_at TIMESTAMP WITH TIME ZONE;
UPDATE statements SET posted_at = created_at WHERE posted_at IS NULL;
ALTER TABLE statements ALTER COLUMN posted_at SET NOT NULL;
ALTER TABLE statements ALTER COLUMN posted_at SET DEFAULT NOW();
CREATE UNIQUE INDEX idx_statements_account_external_id ON statements(account_id, external_id) WHERE external_id IS NOT NULL;
CREATE UNIQUE INDEX idx_statements_unique_reversal ON statements(reference_id) WHERE reference_id IS NOT NULL;
CREATE INDEX idx_statements_account_posted_at ON statements(account_id, posted_at DESC, id DESC);

-- +goose Down
-- WARNING: Dropping these columns is destructive if OFX data has been imported.
DROP INDEX IF EXISTS idx_statements_account_posted_at;
DROP INDEX IF EXISTS idx_statements_unique_reversal;
DROP INDEX IF EXISTS idx_statements_account_external_id;
ALTER TABLE statements DROP COLUMN IF EXISTS posted_at;
ALTER TABLE statements DROP COLUMN IF EXISTS external_id;
