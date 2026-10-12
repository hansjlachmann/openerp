package tables

import (
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/themes"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Theme is a color theme installed from GitHub (business-logic/themerepo). Built-in themes
// (backend/foundation/themes) are not stored; ThemeByCode finds both.
type Theme struct {
	gtables.ThemeBase
}

// NewTheme creates a new Theme instance
func NewTheme() *Theme {
	return &Theme{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *Theme) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.ThemeBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *Theme) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *Theme) OnModify() error {
	return t.Validate()
}

// OnDelete deletes the theme's images. Users who chose the theme fall back to the default
// theme of its mode (handlers/themes.go appearanceOf).
func (t *Theme) OnDelete(db database.Executor, company string) error {
	var assets ThemeAsset
	assets.InitWithDBType(db, "", t.GetDBType())
	assets.SetRange("theme_code", t.Code.String())
	assets.DeleteAll()
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *Theme) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *Theme) Validate() error {
	if err := ftables.CheckRequired(gtables.ThemeTableName, "code", t.Code.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeTableName, "code", string(t.Code), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeTableName, "name", string(t.Name), 100); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeTableName, "repo_url", string(t.Repo_url), 250); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeTableName, "repo_ref", string(t.Repo_ref), 100); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeTableName, "commit_sha", string(t.Commit_sha), 40); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeTableName, "author", string(t.Author), 100); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.ThemeTableName, "installed_by", string(t.Installed_by), 50); err != nil {
		return err
	}

	if _, err := themes.ParseTokens(t.Tokens.String()); err != nil {
		return apperrors.FieldFormat(gtables.ThemeTableName, "tokens", "JSON")
	}
	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// ToTheme returns the stored theme as the API sends it.
func (t *Theme) ToTheme() (themes.Theme, error) {
	tokens, err := themes.ParseTokens(t.Tokens.String())
	if err != nil {
		return themes.Theme{}, err
	}
	return themes.Theme{
		Code: t.Code.String(), Name: t.Name.String(), Mode: tokens.Mode, Source: "github",
		Author: t.Author.String(), RepoURL: t.Repo_url.String(), Tokens: tokens,
		Wallpapers: t.Wallpapers, Version: t.Commit_sha.String(),
	}, nil
}

// InstalledThemes returns the themes installed from GitHub (in code order).
func InstalledThemes(db database.Executor, dbType database.DBType) []themes.Theme {
	var rec Theme
	rec.InitWithDBType(db, "", dbType)
	var out []themes.Theme
	if rec.FindSet() {
		for {
			if th, err := rec.ToTheme(); err == nil {
				out = append(out, th)
			}
			if !rec.Next() {
				break
			}
		}
	}
	return out
}

// ThemeByCode finds a theme a user can choose: built-in or installed.
func ThemeByCode(db database.Executor, dbType database.DBType, code string) (themes.Theme, bool) {
	if th, ok := themes.Builtin(code); ok {
		return th, true
	}
	if db == nil || code == "" {
		return themes.Theme{}, false
	}
	var rec Theme
	rec.InitWithDBType(db, "", dbType)
	if !rec.Get(code) {
		return themes.Theme{}, false
	}
	th, err := rec.ToTheme()
	return th, err == nil
}

// TODO: Add your custom business logic methods here
// Example:
// func (t *Theme) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
