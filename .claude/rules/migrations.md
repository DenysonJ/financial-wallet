---
applies-to: "internal/infrastructure/db/postgres/migration/**"
---
# Migration Rules

- Always include both `-- +goose Up` and `-- +goose Down` sections
- Down migration must be the exact reverse of Up (reversible migrations)
- Use explicit column types, never rely on PostgreSQL defaults
- Add indexes for all foreign keys
- Use `CREATE INDEX CONCURRENTLY` for large tables in production
- Test both up and down migrations locally before committing
- Migration file naming: Goose auto-generates timestamps, do not rename
- Never modify an already-applied migration; create a new one instead
