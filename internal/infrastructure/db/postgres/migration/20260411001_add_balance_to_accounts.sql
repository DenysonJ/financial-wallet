-- +goose Up

ALTER TABLE accounts ADD COLUMN balance BIGINT NOT NULL DEFAULT 0;

-- +goose Down

ALTER TABLE accounts DROP COLUMN balance;
