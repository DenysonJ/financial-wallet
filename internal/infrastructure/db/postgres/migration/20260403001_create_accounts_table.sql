-- +goose Up

-- Accounts table (financial containers: bank accounts, credit cards, cash)
CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('bank_account', 'credit_card', 'cash')),
    description TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id);
CREATE INDEX idx_accounts_user_type ON accounts(user_id, type);
CREATE INDEX idx_accounts_user_active ON accounts(user_id, created_at DESC) WHERE active = true;

-- Seed: account permissions
INSERT INTO permissions (id, name, description, created_at) VALUES
    ('00000000-0000-0000-0000-000000000007', 'account:read', 'Read account data', NOW()),
    ('00000000-0000-0000-0000-000000000008', 'account:write', 'Create accounts', NOW()),
    ('00000000-0000-0000-0000-000000000009', 'account:update', 'Update accounts', NOW()),
    ('00000000-0000-0000-0000-000000000010', 'account:delete', 'Delete accounts', NOW());

-- Seed: admin gets all account permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000007'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000008'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000009'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000010');

-- Seed: user gets read+write+update+delete on own accounts
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000007'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000008'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000009'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000010');

-- +goose Down
DELETE FROM role_permissions WHERE permission_id IN (
    '00000000-0000-0000-0000-000000000007',
    '00000000-0000-0000-0000-000000000008',
    '00000000-0000-0000-0000-000000000009',
    '00000000-0000-0000-0000-000000000010'
);
DELETE FROM permissions WHERE id IN (
    '00000000-0000-0000-0000-000000000007',
    '00000000-0000-0000-0000-000000000008',
    '00000000-0000-0000-0000-000000000009',
    '00000000-0000-0000-0000-000000000010'
);
DROP INDEX IF EXISTS idx_accounts_user_active;
DROP INDEX IF EXISTS idx_accounts_user_type;
DROP INDEX IF EXISTS idx_accounts_user_id;
DROP TABLE IF EXISTS accounts;
