-- +goose Up

-- Categories (system defaults have user_id IS NULL; user-owned have user_id = owner)
CREATE TABLE categories (
    id          UUID PRIMARY KEY,
    user_id     UUID NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(60) NOT NULL,
    type        VARCHAR(10) NOT NULL CHECK (type IN ('credit', 'debit')),
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Unique per (scope, lower(name), type). System defaults coalesce to a sentinel UUID
-- so the same unique index covers both default and user-owned scopes.
CREATE UNIQUE INDEX idx_categories_user_name_type
    ON categories(COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid), LOWER(name), type);

CREATE INDEX idx_categories_user_type ON categories(user_id, type);

-- Tags (system defaults have user_id IS NULL)
CREATE TABLE tags (
    id          UUID PRIMARY KEY,
    user_id     UUID NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(40) NOT NULL,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_tags_user_name
    ON tags(COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::uuid), LOWER(name));

-- Statement <-> Category FK (nullable; RESTRICT prevents deleting a category in use)
ALTER TABLE statements
    ADD COLUMN category_id UUID NULL REFERENCES categories(id) ON DELETE RESTRICT;

CREATE INDEX idx_statements_category ON statements(category_id) WHERE category_id IS NOT NULL;

-- Statement <-> Tag (many-to-many; CASCADE both sides — tags are descriptive metadata)
CREATE TABLE statement_tags (
    statement_id UUID NOT NULL REFERENCES statements(id) ON DELETE CASCADE,
    tag_id       UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (statement_id, tag_id)
);

CREATE INDEX idx_statement_tags_tag ON statement_tags(tag_id);

-- Seed: RBAC permissions for category and tag.
INSERT INTO permissions (id, name, description, created_at) VALUES
    ('00000000-0000-0000-0000-000000000021', 'category:read',  'Read categories',                         NOW()),
    ('00000000-0000-0000-0000-000000000022', 'category:write', 'Create, update and delete categories',    NOW()),
    ('00000000-0000-0000-0000-000000000023', 'tag:read',       'Read tags',                               NOW()),
    ('00000000-0000-0000-0000-000000000024', 'tag:write',      'Create, update and delete tags',          NOW());

-- Seed: admin role gets all four permissions.
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000021'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000022'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000023'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000024');

-- Seed: user role gets read+write (categories and tags are own-resources).
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000021'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000022'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000023'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000024');

-- Seed: default categories (user_id NULL = system).
-- UUID prefix 00000000-0000-0000-0000-2xxxxxxxxxxx; credit = 21xx, debit = 22xx.
INSERT INTO categories (id, user_id, name, type, created_at, updated_at) VALUES
    -- credit (receita)
    ('00000000-0000-0000-0000-210000000001', NULL, 'Salário',          'credit', NOW(), NOW()),
    ('00000000-0000-0000-0000-210000000002', NULL, 'Freelance',        'credit', NOW(), NOW()),
    ('00000000-0000-0000-0000-210000000003', NULL, 'Investimento',     'credit', NOW(), NOW()),
    ('00000000-0000-0000-0000-210000000004', NULL, 'Reembolso',        'credit', NOW(), NOW()),
    ('00000000-0000-0000-0000-210000000005', NULL, 'Presente',         'credit', NOW(), NOW()),
    ('00000000-0000-0000-0000-210000000006', NULL, 'Estorno',          'credit', NOW(), NOW()),
    ('00000000-0000-0000-0000-210000000007', NULL, 'Outros (Receita)', 'credit', NOW(), NOW()),
    -- debit (despesa)
    ('00000000-0000-0000-0000-220000000001', NULL, 'Mercado',          'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000002', NULL, 'Restaurante',      'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000003', NULL, 'Transporte',       'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000004', NULL, 'Moradia',          'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000005', NULL, 'Saúde',            'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000006', NULL, 'Educação',         'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000007', NULL, 'Lazer',            'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000008', NULL, 'Vestuário',        'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-220000000009', NULL, 'Assinaturas',      'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-22000000000a', NULL, 'Impostos',         'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-22000000000b', NULL, 'Transferência',    'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-22000000000c', NULL, 'Estorno',          'debit',  NOW(), NOW()),
    ('00000000-0000-0000-0000-22000000000d', NULL, 'Outros (Despesa)', 'debit',  NOW(), NOW());

-- Seed: default tags (user_id NULL = system).
INSERT INTO tags (id, user_id, name, created_at, updated_at) VALUES
    ('00000000-0000-0000-0000-230000000001', NULL, 'recorrente',    NOW(), NOW()),
    ('00000000-0000-0000-0000-230000000002', NULL, 'reembolsável',  NOW(), NOW()),
    ('00000000-0000-0000-0000-230000000003', NULL, 'pessoal',       NOW(), NOW()),
    ('00000000-0000-0000-0000-230000000004', NULL, 'trabalho',      NOW(), NOW()),
    ('00000000-0000-0000-0000-230000000005', NULL, 'essencial',     NOW(), NOW()),
    ('00000000-0000-0000-0000-230000000006', NULL, 'não-essencial', NOW(), NOW());

-- +goose Down

-- Revert FK + index on statements before dropping categories.
DROP INDEX IF EXISTS idx_statements_category;
ALTER TABLE statements DROP COLUMN IF EXISTS category_id;

-- Drop association table.
DROP INDEX IF EXISTS idx_statement_tags_tag;
DROP TABLE IF EXISTS statement_tags;

-- Drop tags.
DROP INDEX IF EXISTS idx_tags_user_name;
DROP TABLE IF EXISTS tags;

-- Drop categories.
DROP INDEX IF EXISTS idx_categories_user_type;
DROP INDEX IF EXISTS idx_categories_user_name_type;
DROP TABLE IF EXISTS categories;

-- Revert RBAC seeds.
DELETE FROM role_permissions WHERE permission_id IN (
    '00000000-0000-0000-0000-000000000021',
    '00000000-0000-0000-0000-000000000022',
    '00000000-0000-0000-0000-000000000023',
    '00000000-0000-0000-0000-000000000024'
);
DELETE FROM permissions WHERE id IN (
    '00000000-0000-0000-0000-000000000021',
    '00000000-0000-0000-0000-000000000022',
    '00000000-0000-0000-0000-000000000023',
    '00000000-0000-0000-0000-000000000024'
);
