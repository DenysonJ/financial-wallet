-- +goose Up
ALTER TABLE statements ADD COLUMN external_id TEXT;
ALTER TABLE statements ADD COLUMN posted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
CREATE UNIQUE INDEX idx_statements_account_external_id ON statements(account_id, external_id) WHERE external_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_statements_account_external_id;
ALTER TABLE statements DROP COLUMN IF EXISTS posted_at;
ALTER TABLE statements DROP COLUMN IF EXISTS external_id;
