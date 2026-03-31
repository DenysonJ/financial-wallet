-- +goose Up
ALTER TABLE users ADD COLUMN password_hash VARCHAR(255) NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN password_hash;
