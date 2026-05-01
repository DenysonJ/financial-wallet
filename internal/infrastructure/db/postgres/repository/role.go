package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	roledomain "github.com/DenysonJ/financial-wallet/internal/domain/role"
	"github.com/DenysonJ/financial-wallet/pkg/vo"
	"github.com/jmoiron/sqlx"
)

// roleDB é o modelo de banco de dados (Data Model) para Role.
//
// Por que ter um modelo separado da entidade de domínio?
//   - Nomes de colunas podem ser diferentes dos campos da entidade
//   - Permite adicionar campos específicos do banco (audit, soft delete)
//   - Desacopla mudanças de schema de mudanças de domínio
type roleDB struct {
	ID          string    `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

func (r *roleDB) toRole() (*roledomain.Role, error) {
	id, parseErr := vo.ParseID(r.ID)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing ID: %w", parseErr)
	}

	return &roledomain.Role{
		ID:          id,
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}, nil
}

func fromDomainRole(r *roledomain.Role) roleDB {
	return roleDB{
		ID:          r.ID.String(),
		Name:        r.Name,
		Description: r.Description,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

// RoleRepository implementa a interface Repository para Role.
type RoleRepository struct {
	writer *sqlx.DB
	reader *sqlx.DB
}

// NewRoleRepository cria uma nova instância do repositório.
func NewRoleRepository(writer, reader *sqlx.DB) *RoleRepository {
	return &RoleRepository{writer: writer, reader: reader}
}

func (r *RoleRepository) Create(ctx context.Context, role *roledomain.Role) error {
	query := `
		INSERT INTO roles (
			id, name, description, created_at, updated_at
		) VALUES (
			:id, :name, :description, :created_at, :updated_at
		)
	`

	dbModel := fromDomainRole(role)
	_, execErr := r.writer.NamedExecContext(ctx, query, dbModel)
	return execErr
}

func (r *RoleRepository) FindByName(ctx context.Context, name string) (*roledomain.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE name = $1
	`

	var dbModel roleDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, name)
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, roledomain.ErrRoleNotFound
		}
		return nil, selectErr
	}

	return dbModel.toRole()
}

func (r *RoleRepository) List(ctx context.Context, filter roledomain.ListFilter) (*roledomain.ListResult, error) {
	filter.Normalize()

	// Build dynamic query with filters
	args := make(map[string]interface{})

	whereClause := ""
	if filter.Name != "" {
		whereClause = "WHERE name ILIKE :name"
		args["name"] = "%" + escapeILIKE(filter.Name) + "%"
	}

	// Wrap COUNT + SELECT in a read-only transaction for consistent pagination.
	// Without a transaction, rows could be inserted/deleted between the two queries,
	// causing total count to be inconsistent with the returned data.
	tx, txErr := r.reader.BeginTxx(ctx, &sql.TxOptions{ReadOnly: true})
	if txErr != nil {
		return nil, fmt.Errorf("beginning read transaction: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM roles %s", whereClause)

	countQuery, countArgs, namedErr := sqlx.Named(countQuery, args)
	if namedErr != nil {
		return nil, namedErr
	}
	countQuery = tx.Rebind(countQuery)

	var total int
	countErr := tx.GetContext(ctx, &total, countQuery, countArgs...)
	if countErr != nil {
		return nil, countErr
	}

	// Paginated data query
	args["limit"] = filter.Limit
	args["offset"] = filter.Offset()

	dataQuery := fmt.Sprintf(`
		SELECT id, name, description, created_at, updated_at
		FROM roles
		%s
		ORDER BY created_at DESC
		LIMIT :limit OFFSET :offset
	`, whereClause)

	dataQuery, dataArgs, dataNamedErr := sqlx.Named(dataQuery, args)
	if dataNamedErr != nil {
		return nil, dataNamedErr
	}
	dataQuery = tx.Rebind(dataQuery)

	var dbModels []roleDB
	selectErr := tx.SelectContext(ctx, &dbModels, dataQuery, dataArgs...)
	if selectErr != nil {
		return nil, selectErr
	}

	// Commit the read-only transaction (also valid to let defer Rollback handle it,
	// but explicit commit is cleaner for read-only transactions).
	commitErr := tx.Commit()
	if commitErr != nil {
		return nil, fmt.Errorf("committing read transaction: %w", commitErr)
	}

	// Convert to domain roles
	roles := make([]*roledomain.Role, 0, len(dbModels))
	for i := range dbModels {
		role, convertErr := dbModels[i].toRole()
		if convertErr != nil {
			return nil, convertErr
		}
		roles = append(roles, role)
	}

	return &roledomain.ListResult{
		Roles: roles,
		Total: total,
		Page:  filter.Page,
		Limit: filter.Limit,
	}, nil
}

func (r *RoleRepository) FindByID(ctx context.Context, id vo.ID) (*roledomain.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE id = $1
	`

	var dbModel roleDB
	selectErr := r.reader.GetContext(ctx, &dbModel, query, id.String())
	if selectErr != nil {
		if errors.Is(selectErr, sql.ErrNoRows) {
			return nil, roledomain.ErrRoleNotFound
		}
		return nil, selectErr
	}

	return dbModel.toRole()
}

func (r *RoleRepository) AssignRole(ctx context.Context, userID, roleID vo.ID) error {
	query := `
		INSERT INTO user_roles (user_id, role_id, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	result, execErr := r.writer.ExecContext(ctx, query, userID.String(), roleID.String(), time.Now())
	if execErr != nil {
		return execErr
	}

	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}

	if rowsAffected == 0 {
		return roledomain.ErrRoleAlreadyAssigned
	}

	return nil
}

func (r *RoleRepository) RevokeRole(ctx context.Context, userID, roleID vo.ID) error {
	query := `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = $2
	`

	result, execErr := r.writer.ExecContext(ctx, query, userID.String(), roleID.String())
	if execErr != nil {
		return execErr
	}

	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}

	if rowsAffected == 0 {
		return roledomain.ErrRoleNotAssigned
	}

	return nil
}

func (r *RoleRepository) GetUserPermissions(ctx context.Context, userID vo.ID) ([]string, error) {
	query := `
		SELECT DISTINCT p.name
		FROM user_roles ur
		JOIN role_permissions rp ON ur.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE ur.user_id = $1
		ORDER BY p.name
	`

	var permissions []string
	selectErr := r.reader.SelectContext(ctx, &permissions, query, userID.String())
	if selectErr != nil {
		return nil, selectErr
	}

	return permissions, nil
}

func (r *RoleRepository) GetUserRoles(ctx context.Context, userID vo.ID) ([]string, error) {
	query := `
		SELECT DISTINCT rl.name
		FROM user_roles ur
		JOIN roles rl ON ur.role_id = rl.id
		WHERE ur.user_id = $1
		ORDER BY rl.name
	`

	var roles []string
	selectErr := r.reader.SelectContext(ctx, &roles, query, userID.String())
	if selectErr != nil {
		return nil, selectErr
	}

	return roles, nil
}

// userPermsRolesRow models a row from the combined GetUserPermissionsAndRoles
// query: kind is 'p' for permission, 'r' for role.
type userPermsRolesRow struct {
	Kind string `db:"kind"`
	Name string `db:"name"`
}

// GetUserPermissionsAndRoles returns permissions + roles in a single UNION ALL
// query, saving one RTT on cache-miss vs the two-query path.
func (r *RoleRepository) GetUserPermissionsAndRoles(ctx context.Context, userID vo.ID) (perms, roles []string, err error) {
	query := `
		SELECT 'p' AS kind, p.name AS name
		FROM user_roles ur
		JOIN role_permissions rp ON ur.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE ur.user_id = $1
		UNION ALL
		SELECT 'r' AS kind, rl.name AS name
		FROM user_roles ur
		JOIN roles rl ON ur.role_id = rl.id
		WHERE ur.user_id = $1
	`

	var rows []userPermsRolesRow
	selectErr := r.reader.SelectContext(ctx, &rows, query, userID.String())
	if selectErr != nil {
		return nil, nil, selectErr
	}

	permsSet := make(map[string]struct{})
	rolesSet := make(map[string]struct{})
	for _, row := range rows {
		switch row.Kind {
		case "p":
			permsSet[row.Name] = struct{}{}
		case "r":
			rolesSet[row.Name] = struct{}{}
		}
	}

	perms = make([]string, 0, len(permsSet))
	for name := range permsSet {
		perms = append(perms, name)
	}
	roles = make([]string, 0, len(rolesSet))
	for name := range rolesSet {
		roles = append(roles, name)
	}
	sort.Strings(perms)
	sort.Strings(roles)
	return perms, roles, nil
}

func (r *RoleRepository) Delete(ctx context.Context, id vo.ID) error {
	query := `
		DELETE FROM roles
		WHERE id = $1
	`

	result, execErr := r.writer.ExecContext(ctx, query, id.String())
	if execErr != nil {
		return execErr
	}

	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return rowsErr
	}

	if rowsAffected == 0 {
		return roledomain.ErrRoleNotFound
	}

	return nil
}
