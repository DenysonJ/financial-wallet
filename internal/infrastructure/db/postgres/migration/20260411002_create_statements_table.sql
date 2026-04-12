-- +goose Up

-- Statements table (append-only financial ledger)
CREATE TABLE statements (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id),
    type VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')),
    amount BIGINT NOT NULL CHECK (amount > 0),
    description TEXT NOT NULL DEFAULT '',
    reference_id UUID REFERENCES statements(id),
    balance_after BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE INDEX idx_statements_account_id ON statements(account_id, created_at DESC);
CREATE INDEX idx_statements_reference_id ON statements(reference_id) WHERE reference_id IS NOT NULL;
CREATE INDEX idx_statements_account_type ON statements(account_id, type);

-- Seed: statement permissions
INSERT INTO permissions (id, name, description) VALUES
    ('00000000-0000-0000-0000-000000000011', 'statement:read', 'Read statement data'),
    ('00000000-0000-0000-0000-000000000012', 'statement:write', 'Create statements');

-- Seed: admin gets all statement permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000011'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000012');

-- Seed: user gets read+write on own statements
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000011'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000012');

-- +goose Down

DELETE FROM role_permissions WHERE permission_id IN (
    '00000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000012'
);
DELETE FROM permissions WHERE id IN (
    '00000000-0000-0000-0000-000000000011',
    '00000000-0000-0000-0000-000000000012'
);
DROP INDEX IF EXISTS idx_statements_account_type;
DROP INDEX IF EXISTS idx_statements_reference_id;
DROP INDEX IF EXISTS idx_statements_account_id;
DROP TABLE IF EXISTS statements;
