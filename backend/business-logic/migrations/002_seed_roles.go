package migrations

import (
	"fmt"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	fmigrations "github.com/hansjlachmann/openerp/backend/foundation/migrations"
)

func init() {
	Register(&Migration002SeedRoles{})
}

// Migration002SeedRoles seeds default User_Role and Permission records
type Migration002SeedRoles struct{}

func (m *Migration002SeedRoles) Version() int {
	return 2
}

func (m *Migration002SeedRoles) Name() string {
	return "seed_default_roles"
}

func (m *Migration002SeedRoles) Description() string {
	return "Seed SUPER and READER default roles with permissions"
}

func (m *Migration002SeedRoles) Up(ctx *fmigrations.Context) error {
	// Insert SUPER role (bypasses all permission checks via IsSuper flag)
	if err := m.insertRole(ctx, "SUPER", "Full access - bypasses all permission checks"); err != nil {
		return fmt.Errorf("failed to insert SUPER role: %w", err)
	}

	// Insert READER role (read-only access to all tables)
	if err := m.insertRole(ctx, "READER", "Read-only access to all tables"); err != nil {
		return fmt.Errorf("failed to insert READER role: %w", err)
	}

	// READER gets read permission on all business tables
	readerTables := []string{
		"Customer",
		"Customer_ledger_entry",
		"Payment_terms",
		"Job_Queue",
		"Job_Queue_Entry",
		"User",
		"User_Role",
		"User_Member",
		"Permission",
		"Language",
		"Menu",
		"Company",
	}

	for _, table := range readerTables {
		if err := m.insertPermission(ctx, "READER", table, true, false, false, false); err != nil {
			return fmt.Errorf("failed to insert READER permission for %s: %w", table, err)
		}
	}

	return nil
}

func (m *Migration002SeedRoles) insertRole(ctx *fmigrations.Context, code, description string) error {
	query := `INSERT INTO "User_Role" (code, description) VALUES (%s, %s)`
	p1, p2 := m.placeholders(ctx, 1, 2)
	return ctx.ExecuteSQL(fmt.Sprintf(query, p1, p2), code, description)
}

func (m *Migration002SeedRoles) insertPermission(ctx *fmigrations.Context, roleID, tableName string, canRead, canInsert, canModify, canDelete bool) error {
	query := `INSERT INTO "Permission" (role_id, table_name, can_read, can_insert, can_modify, can_delete) VALUES (%s, %s, %s, %s, %s, %s)`
	p := make([]string, 6)
	for i := range p {
		p[i] = m.placeholder(ctx, i+1)
	}
	return ctx.ExecuteSQL(fmt.Sprintf(query, p[0], p[1], p[2], p[3], p[4], p[5]), roleID, tableName, canRead, canInsert, canModify, canDelete)
}

func (m *Migration002SeedRoles) placeholder(ctx *fmigrations.Context, n int) string {
	if ctx.DBType == database.DBTypePostgres {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

func (m *Migration002SeedRoles) placeholders(ctx *fmigrations.Context, nums ...int) (string, string) {
	results := make([]string, len(nums))
	for i, n := range nums {
		results[i] = m.placeholder(ctx, n)
	}
	return results[0], results[1]
}
