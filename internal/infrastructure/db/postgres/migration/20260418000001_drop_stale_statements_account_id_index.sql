-- +goose NO TRANSACTION
-- +goose Up
-- Drop the stale idx_statements_account_id: it orders on created_at, but every
-- List query orders on (posted_at DESC, id DESC). Coverage is provided by
-- idx_statements_account_posted_at from migration 20260416002257.
DROP INDEX CONCURRENTLY IF EXISTS idx_statements_account_id;

-- +goose Down
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_statements_account_id ON statements(account_id, created_at DESC);
