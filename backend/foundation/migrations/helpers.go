package migrations

import (
	"fmt"
	"strings"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
)

// RenameColumn renames a column in a table
// Note: PostgreSQL supports ALTER TABLE RENAME COLUMN directly
// SQLite requires recreating the table (not implemented here - use raw SQL)
func (ctx *Context) RenameColumn(tableName, oldName, newName string) error {
	switch ctx.DBType {
	case database.DBTypePostgres:
		sql := fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN %s TO %s`, tableName, oldName, newName)
		return ctx.ExecuteSQL(sql)
	default:
		// SQLite doesn't support RENAME COLUMN in older versions
		// For SQLite 3.25.0+, this works:
		sql := fmt.Sprintf(`ALTER TABLE "%s" RENAME COLUMN %s TO %s`, tableName, oldName, newName)
		return ctx.ExecuteSQL(sql)
	}
}

// DropColumn removes a column from a table
// Note: PostgreSQL supports ALTER TABLE DROP COLUMN directly
// SQLite doesn't support DROP COLUMN (requires table recreation)
func (ctx *Context) DropColumn(tableName, columnName string) error {
	switch ctx.DBType {
	case database.DBTypePostgres:
		sql := fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN IF EXISTS %s`, tableName, columnName)
		return ctx.ExecuteSQL(sql)
	default:
		// SQLite 3.35.0+ supports DROP COLUMN
		sql := fmt.Sprintf(`ALTER TABLE "%s" DROP COLUMN %s`, tableName, columnName)
		return ctx.ExecuteSQL(sql)
	}
}

// AddColumn adds a new column to a table
func (ctx *Context) AddColumn(tableName, columnName, definition string) error {
	sql := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN %s %s`, tableName, columnName, definition)
	return ctx.ExecuteSQL(sql)
}

// AddColumnIfNotExists adds a column only if it doesn't exist
func (ctx *Context) AddColumnIfNotExists(tableName, columnName, definition string) error {
	exists, err := ctx.ColumnExists(tableName, columnName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return ctx.AddColumn(tableName, columnName, definition)
}

// ColumnExists checks if a column exists in a table
func (ctx *Context) ColumnExists(tableName, columnName string) (bool, error) {
	var count int

	switch ctx.DBType {
	case database.DBTypePostgres:
		err := ctx.tx.QueryRow(`
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2
		`, tableName, strings.ToLower(columnName)).Scan(&count)
		if err != nil {
			return false, err
		}
	default:
		rows, err := ctx.tx.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, tableName))
		if err != nil {
			return false, err
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var dfltValue interface{}
			var pk int

			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				return false, err
			}
			if strings.EqualFold(name, columnName) {
				return true, nil
			}
		}
		return false, nil
	}

	return count > 0, nil
}

// TableExists checks if a table exists
func (ctx *Context) TableExists(tableName string) (bool, error) {
	var name string

	switch ctx.DBType {
	case database.DBTypePostgres:
		err := ctx.tx.QueryRow(`
			SELECT table_name FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = $1
		`, tableName).Scan(&name)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				return false, nil
			}
			return false, err
		}
	default:
		err := ctx.tx.QueryRow(`
			SELECT name FROM sqlite_master
			WHERE type='table' AND name=?
		`, tableName).Scan(&name)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				return false, nil
			}
			return false, err
		}
	}

	return true, nil
}

// CreateIndex creates an index on a table
func (ctx *Context) CreateIndex(indexName, tableName string, columns ...string) error {
	sql := fmt.Sprintf(`CREATE INDEX IF NOT EXISTS "%s" ON "%s" (%s)`,
		indexName, tableName, strings.Join(columns, ", "))
	return ctx.ExecuteSQL(sql)
}

// DropIndex removes an index
func (ctx *Context) DropIndex(indexName string) error {
	sql := fmt.Sprintf(`DROP INDEX IF EXISTS "%s"`, indexName)
	return ctx.ExecuteSQL(sql)
}

// UpdateData updates data in a table with a WHERE clause
func (ctx *Context) UpdateData(tableName, setClause, whereClause string, args ...interface{}) error {
	sql := fmt.Sprintf(`UPDATE "%s" SET %s WHERE %s`, tableName, setClause, whereClause)
	return ctx.ExecuteSQL(sql, args...)
}

// DeleteData deletes data from a table with a WHERE clause
func (ctx *Context) DeleteData(tableName, whereClause string, args ...interface{}) error {
	sql := fmt.Sprintf(`DELETE FROM "%s" WHERE %s`, tableName, whereClause)
	return ctx.ExecuteSQL(sql, args...)
}

// CopyColumnData copies data from one column to another (useful before dropping)
func (ctx *Context) CopyColumnData(tableName, sourceColumn, targetColumn string) error {
	sql := fmt.Sprintf(`UPDATE "%s" SET %s = %s`, tableName, targetColumn, sourceColumn)
	return ctx.ExecuteSQL(sql)
}

// SetColumnDefault sets a default value for existing NULL values
func (ctx *Context) SetColumnDefault(tableName, columnName string, defaultValue interface{}) error {
	var sql string
	switch v := defaultValue.(type) {
	case string:
		sql = fmt.Sprintf(`UPDATE "%s" SET %s = '%s' WHERE %s IS NULL`, tableName, columnName, v, columnName)
	default:
		sql = fmt.Sprintf(`UPDATE "%s" SET %s = %v WHERE %s IS NULL`, tableName, columnName, v, columnName)
	}
	return ctx.ExecuteSQL(sql)
}

// ChangeColumnType changes a column's data type (PostgreSQL only, requires data compatibility)
func (ctx *Context) ChangeColumnType(tableName, columnName, newType string) error {
	if ctx.DBType != database.DBTypePostgres {
		return fmt.Errorf("ChangeColumnType is only supported on PostgreSQL")
	}

	sql := fmt.Sprintf(`ALTER TABLE "%s" ALTER COLUMN %s TYPE %s`, tableName, columnName, newType)
	return ctx.ExecuteSQL(sql)
}
