package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/manimovassagh/rampart/internal/model"
)

// ListRoles returns a paginated, searchable list of roles for an org.
func (db *DB) ListRoles(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.Role, int, error) {
	where := []string{"org_id = $1"}
	args := []any{orgID}
	paramIdx := 2

	if search != "" {
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", paramIdx, paramIdx))
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM roles WHERE %s", whereClause)
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting roles: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, org_id, name, description, builtin, created_at, updated_at
		FROM roles
		WHERE %s
		ORDER BY builtin DESC, name ASC
		LIMIT $%d OFFSET $%d`, whereClause, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing roles: %w", err)
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		var r model.Role
		if err := rows.Scan(&r.ID, &r.OrgID, &r.Name, &r.Description, &r.Builtin, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning role row: %w", err)
		}
		roles = append(roles, &r)
	}

	return roles, total, nil
}

// GetRoleByID retrieves a role by its UUID.
func (db *DB) GetRoleByID(ctx context.Context, id uuid.UUID) (*model.Role, error) {
	query := `SELECT id, org_id, name, description, builtin, created_at, updated_at FROM roles WHERE id = $1`

	var r model.Role
	err := db.Pool.QueryRow(ctx, query, id).Scan(&r.ID, &r.OrgID, &r.Name, &r.Description, &r.Builtin, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying role by id: %w", err)
	}
	return &r, nil
}

// CreateRole inserts a new role.
func (db *DB) CreateRole(ctx context.Context, role *model.Role) (*model.Role, error) {
	query := `
		INSERT INTO roles (org_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, org_id, name, description, builtin, created_at, updated_at`

	var r model.Role
	err := db.Pool.QueryRow(ctx, query, role.OrgID, role.Name, role.Description).Scan(
		&r.ID, &r.OrgID, &r.Name, &r.Description, &r.Builtin, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting role: %w", err)
	}
	return &r, nil
}

// UpdateRole updates mutable fields on a role.
func (db *DB) UpdateRole(ctx context.Context, id uuid.UUID, req *model.UpdateRoleRequest) (*model.Role, error) {
	query := `
		UPDATE roles
		SET name = COALESCE(NULLIF($2, ''), name),
		    description = $3,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, org_id, name, description, builtin, created_at, updated_at`

	var r model.Role
	err := db.Pool.QueryRow(ctx, query, id, req.Name, req.Description).Scan(
		&r.ID, &r.OrgID, &r.Name, &r.Description, &r.Builtin, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("updating role: %w", err)
	}
	return &r, nil
}

// DeleteRole removes a role by ID. Rejects deletion of builtin roles.
func (db *DB) DeleteRole(ctx context.Context, id uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, "DELETE FROM roles WHERE id = $1 AND builtin = false", id)
	if err != nil {
		return fmt.Errorf("deleting role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("role not found or is builtin")
	}
	return nil
}

// CountRoles returns the total number of roles in an org.
func (db *DB) CountRoles(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM roles WHERE org_id = $1", orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting roles: %w", err)
	}
	return count, nil
}

// CountRoleUsers returns the number of users assigned to a role.
func (db *DB) CountRoleUsers(ctx context.Context, roleID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_roles WHERE role_id = $1", roleID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting role users: %w", err)
	}
	return count, nil
}

// AssignRole assigns a role to a user.
func (db *DB) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		userID, roleID)
	if err != nil {
		return fmt.Errorf("assigning role: %w", err)
	}
	return nil
}

// UnassignRole removes a role from a user.
func (db *DB) UnassignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2", userID, roleID)
	if err != nil {
		return fmt.Errorf("unassigning role: %w", err)
	}
	return nil
}

// GetUserRoles returns all roles assigned to a user.
func (db *DB) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*model.Role, error) {
	query := `
		SELECT r.id, r.org_id, r.name, r.description, r.builtin, r.created_at, r.updated_at
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name`

	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user roles: %w", err)
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		var r model.Role
		if err := rows.Scan(&r.ID, &r.OrgID, &r.Name, &r.Description, &r.Builtin, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user role row: %w", err)
		}
		roles = append(roles, &r)
	}
	return roles, nil
}

// GetUserRoleNames returns the role names for a user (for JWT claims).
func (db *DB) GetUserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT r.name
		FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name`

	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user role names: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning role name: %w", err)
		}
		names = append(names, name)
	}
	return names, nil
}

// GetRoleUsers returns all users assigned to a role.
func (db *DB) GetRoleUsers(ctx context.Context, roleID uuid.UUID) ([]*model.UserRoleAssignment, error) {
	query := `
		SELECT u.id, u.username, u.email, ur.assigned_at
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ur.role_id = $1
		ORDER BY ur.assigned_at DESC`

	rows, err := db.Pool.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("listing role users: %w", err)
	}
	defer rows.Close()

	var assignments []*model.UserRoleAssignment
	for rows.Next() {
		var a model.UserRoleAssignment
		if err := rows.Scan(&a.UserID, &a.Username, &a.Email, &a.AssignedAt); err != nil {
			return nil, fmt.Errorf("scanning role user row: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, nil
}
