---
name: migrate
description: Goose migration management (create, up, down, status)
user-invocable: true
---

# /migrate <create|up|down|status>

Manages database migrations using Goose.

## Actions

### create
```
/migrate create <name>
```
1. Run `make migrate-create NAME=<name>`
2. Edit the generated SQL file
3. Ensure both `-- +goose Up` and `-- +goose Down` sections exist
4. Down must be the exact reverse of Up

### up
```
/migrate up
```
1. Run `make migrate-up`
2. Verify with `make migrate-status`

### down
```
/migrate down
```
1. Run `make migrate-down`
2. Verify with `make migrate-status`

### status
```
/migrate status
```
1. Run `make migrate-status`

## Migration Rules

- Always include both `-- +goose Up` and `-- +goose Down`
- Use explicit column types
- Add indexes for all foreign keys
- Use `CREATE INDEX CONCURRENTLY` for large tables
- Never modify an already-applied migration
- Test both up and down locally before committing
- File naming: Goose auto-generates timestamps, do not rename
