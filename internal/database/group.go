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

// ListGroups returns a paginated, searchable list of groups for an org.
func (db *DB) ListGroups(ctx context.Context, orgID uuid.UUID, search string, limit, offset int) ([]*model.Group, int, error) {
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
	if err := db.Pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM groups WHERE %s", whereClause), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting groups: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT id, org_id, name, description, created_at, updated_at
		FROM groups
		WHERE %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d`, whereClause, paramIdx, paramIdx+1)
	args = append(args, limit, offset)

	rows, err := db.Pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("listing groups: %w", err)
	}
	defer rows.Close()

	var groups []*model.Group
	for rows.Next() {
		var g model.Group
		if err := rows.Scan(&g.ID, &g.OrgID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning group row: %w", err)
		}
		groups = append(groups, &g)
	}
	return groups, total, nil
}

// GetGroupByID retrieves a group by its UUID.
func (db *DB) GetGroupByID(ctx context.Context, id uuid.UUID) (*model.Group, error) {
	var g model.Group
	err := db.Pool.QueryRow(ctx,
		"SELECT id, org_id, name, description, created_at, updated_at FROM groups WHERE id = $1", id,
	).Scan(&g.ID, &g.OrgID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("querying group by id: %w", err)
	}
	return &g, nil
}

// CreateGroup inserts a new group.
func (db *DB) CreateGroup(ctx context.Context, group *model.Group) (*model.Group, error) {
	var g model.Group
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO groups (org_id, name, description) VALUES ($1, $2, $3)
		 RETURNING id, org_id, name, description, created_at, updated_at`,
		group.OrgID, group.Name, group.Description,
	).Scan(&g.ID, &g.OrgID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting group: %w", err)
	}
	return &g, nil
}

// UpdateGroup updates mutable fields on a group.
func (db *DB) UpdateGroup(ctx context.Context, id uuid.UUID, req *model.UpdateGroupRequest) (*model.Group, error) {
	var g model.Group
	err := db.Pool.QueryRow(ctx,
		`UPDATE groups SET name = COALESCE(NULLIF($2, ''), name), description = $3, updated_at = now()
		 WHERE id = $1
		 RETURNING id, org_id, name, description, created_at, updated_at`,
		id, req.Name, req.Description,
	).Scan(&g.ID, &g.OrgID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("updating group: %w", err)
	}
	return &g, nil
}

// DeleteGroup removes a group by ID.
func (db *DB) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	tag, err := db.Pool.Exec(ctx, "DELETE FROM groups WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("group not found")
	}
	return nil
}

// CountGroups returns the total number of groups in an org.
func (db *DB) CountGroups(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM groups WHERE org_id = $1", orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting groups: %w", err)
	}
	return count, nil
}

// CountGroupMembers returns the number of members in a group.
func (db *DB) CountGroupMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM user_groups WHERE group_id = $1", groupID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting group members: %w", err)
	}
	return count, nil
}

// CountGroupRoles returns the number of roles assigned to a group.
func (db *DB) CountGroupRoles(ctx context.Context, groupID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM group_roles WHERE group_id = $1", groupID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting group roles: %w", err)
	}
	return count, nil
}

// AddUserToGroup adds a user to a group.
func (db *DB) AddUserToGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		userID, groupID)
	if err != nil {
		return fmt.Errorf("adding user to group: %w", err)
	}
	return nil
}

// RemoveUserFromGroup removes a user from a group.
func (db *DB) RemoveUserFromGroup(ctx context.Context, userID, groupID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM user_groups WHERE user_id = $1 AND group_id = $2", userID, groupID)
	if err != nil {
		return fmt.Errorf("removing user from group: %w", err)
	}
	return nil
}

// GetGroupMembers returns all members of a group.
func (db *DB) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]*model.GroupMember, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT u.id, u.username, u.email, ug.added_at
		 FROM users u JOIN user_groups ug ON ug.user_id = u.id
		 WHERE ug.group_id = $1 ORDER BY ug.added_at DESC`, groupID)
	if err != nil {
		return nil, fmt.Errorf("listing group members: %w", err)
	}
	defer rows.Close()

	var members []*model.GroupMember
	for rows.Next() {
		var m model.GroupMember
		if err := rows.Scan(&m.UserID, &m.Username, &m.Email, &m.AddedAt); err != nil {
			return nil, fmt.Errorf("scanning group member: %w", err)
		}
		members = append(members, &m)
	}
	return members, nil
}

// AssignRoleToGroup assigns a role to a group.
func (db *DB) AssignRoleToGroup(ctx context.Context, groupID, roleID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO group_roles (group_id, role_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		groupID, roleID)
	if err != nil {
		return fmt.Errorf("assigning role to group: %w", err)
	}
	return nil
}

// UnassignRoleFromGroup removes a role from a group.
func (db *DB) UnassignRoleFromGroup(ctx context.Context, groupID, roleID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM group_roles WHERE group_id = $1 AND role_id = $2", groupID, roleID)
	if err != nil {
		return fmt.Errorf("unassigning role from group: %w", err)
	}
	return nil
}

// GetGroupRoles returns all roles assigned to a group.
func (db *DB) GetGroupRoles(ctx context.Context, groupID uuid.UUID) ([]*model.GroupRoleAssignment, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT r.id, r.name, r.description
		 FROM roles r JOIN group_roles gr ON gr.role_id = r.id
		 WHERE gr.group_id = $1 ORDER BY r.name`, groupID)
	if err != nil {
		return nil, fmt.Errorf("listing group roles: %w", err)
	}
	defer rows.Close()

	var assignments []*model.GroupRoleAssignment
	for rows.Next() {
		var a model.GroupRoleAssignment
		if err := rows.Scan(&a.RoleID, &a.RoleName, &a.Description); err != nil {
			return nil, fmt.Errorf("scanning group role: %w", err)
		}
		assignments = append(assignments, &a)
	}
	return assignments, nil
}

// GetEffectiveUserRoles returns role names that include both direct and group-inherited roles.
func (db *DB) GetEffectiveUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT r.name FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		UNION
		SELECT DISTINCT r.name FROM roles r
		JOIN group_roles gr ON gr.role_id = r.id
		JOIN user_groups ug ON ug.group_id = gr.group_id
		WHERE ug.user_id = $1
		ORDER BY name`

	rows, err := db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("getting effective user roles: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning effective role name: %w", err)
		}
		names = append(names, name)
	}
	return names, nil
}

// GetUserGroups returns all groups a user belongs to.
func (db *DB) GetUserGroups(ctx context.Context, userID uuid.UUID) ([]*model.Group, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT g.id, g.org_id, g.name, g.description, g.created_at, g.updated_at
		 FROM groups g JOIN user_groups ug ON ug.group_id = g.id
		 WHERE ug.user_id = $1 ORDER BY g.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing user groups: %w", err)
	}
	defer rows.Close()

	var groups []*model.Group
	for rows.Next() {
		var g model.Group
		if err := rows.Scan(&g.ID, &g.OrgID, &g.Name, &g.Description, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning user group: %w", err)
		}
		groups = append(groups, &g)
	}
	return groups, nil
}
