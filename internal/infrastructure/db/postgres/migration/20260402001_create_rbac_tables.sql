-- +goose Up

-- Permissions (granular RBAC permissions)
CREATE TABLE permissions (
    id UUID PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Role-Permission junction (which permissions each role has). The composite
-- primary key (role_id, permission_id) already provides a B-tree usable for
-- `WHERE role_id = $1` lookups, so no additional single-column index is needed.
CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- User-Role junction (which roles each user has)
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- Seed: permissions
INSERT INTO permissions (id, name, description, created_at) VALUES
    ('00000000-0000-0000-0000-000000000001', 'user:read', 'Read user data', NOW()),
    ('00000000-0000-0000-0000-000000000002', 'user:write', 'Create and update users', NOW()),
    ('00000000-0000-0000-0000-000000000003', 'user:delete', 'Delete users', NOW()),
    ('00000000-0000-0000-0000-000000000004', 'role:read', 'Read roles', NOW()),
    ('00000000-0000-0000-0000-000000000005', 'role:write', 'Create, update, assign and revoke roles', NOW()),
    ('00000000-0000-0000-0000-000000000006', 'role:delete', 'Delete roles', NOW());

-- Seed: roles (admin and user)
INSERT INTO roles (id, name, description, created_at, updated_at) VALUES
    ('00000000-0000-0000-0000-100000000001', 'admin', 'Full access to all resources', NOW(), NOW()),
    ('00000000-0000-0000-0000-100000000002', 'user', 'Access to own resources only', NOW(), NOW())
ON CONFLICT (name) DO NOTHING;

-- Seed: admin gets all permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000001'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000002'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000003'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000004'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000005'),
    ('00000000-0000-0000-0000-100000000001', '00000000-0000-0000-0000-000000000006');

-- Seed: user gets read+write on own data
INSERT INTO role_permissions (role_id, permission_id) VALUES
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000001'),
    ('00000000-0000-0000-0000-100000000002', '00000000-0000-0000-0000-000000000002');

-- +goose Down
DELETE FROM role_permissions WHERE role_id IN (
    '00000000-0000-0000-0000-100000000001',
    '00000000-0000-0000-0000-100000000002'
);
DELETE FROM user_roles WHERE role_id IN (
    '00000000-0000-0000-0000-100000000001',
    '00000000-0000-0000-0000-100000000002'
);
DELETE FROM roles WHERE id IN (
    '00000000-0000-0000-0000-100000000001',
    '00000000-0000-0000-0000-100000000002'
);
DELETE FROM permissions WHERE id IN (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000004',
    '00000000-0000-0000-0000-000000000005',
    '00000000-0000-0000-0000-000000000006'
);
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
