-- +goose Up

-- No CHECK on balance: credit-card / overdraft accounts can be negative.
-- Per-type validation lives in the domain layer.
ALTER TABLE accounts ADD COLUMN balance BIGINT NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE accounts DROP COLUMN balance;
