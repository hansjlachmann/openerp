package tables

import (
	"time"

	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/themes"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// Theme is a color theme installed from GitHub (business-logic/themerepo). The files are
// stored once; each user sees the themes they added (User_Theme). Built-in themes
// (backend/foundation/themes) are not stored and every user has them.
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

// OnDelete deletes the theme's images and removes it from every user's themes. Users who
// chose the theme fall back to the default theme of its mode (handlers/themes.go).
func (t *Theme) OnDelete(db database.Executor, company string) error {
	var links UserTheme
	links.InitWithDBType(db, "", t.GetDBType())
	links.SetRange("theme_code", t.Code.String())
	links.DeleteAll()

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

// UserThemes returns the installed themes the user has added (in code order).
func UserThemes(db database.Executor, dbType database.DBType, userID string) []themes.Theme {
	var links UserTheme
	links.InitWithDBType(db, "", dbType)
	links.SetRange("user_id", userID)
	var codes []string
	if links.FindSet() {
		for {
			codes = append(codes, links.Theme_code.String())
			if !links.Next() {
				break
			}
		}
	}
	var out []themes.Theme
	for _, code := range codes {
		if th, ok := ThemeByCode(db, dbType, code); ok && th.Source == "github" {
			out = append(out, th)
		}
	}
	return out
}

// UserHasTheme reports whether the user has added the installed theme.
func UserHasTheme(db database.Executor, dbType database.DBType, userID, code string) bool {
	var link UserTheme
	link.InitWithDBType(db, "", dbType)
	return link.GetByPK(types.NewCode(userID), types.NewCode(code))
}

// AddUserTheme adds an installed theme to the user's themes (nothing when it is there).
func AddUserTheme(db database.Executor, dbType database.DBType, userID, code string) error {
	if UserHasTheme(db, dbType, userID, code) {
		return nil
	}
	var link UserTheme
	link.InitWithDBType(db, "", dbType)
	link.User_id = types.NewCode(userID)
	link.Theme_code = types.NewCode(code)
	link.Added_at = types.NewDateTimeFromTime(time.Now())
	if !link.Insert(true) {
		if err := link.TriggerError(); err != nil {
			return err
		}
		return apperrors.InsertFailed(gtables.UserThemeTableName)
	}
	return nil
}

// UserThemeByCode finds a theme the user can choose: built-in, or installed and added by
// the user.
func UserThemeByCode(db database.Executor, dbType database.DBType, userID, code string) (themes.Theme, bool) {
	if th, ok := themes.Builtin(code); ok {
		return th, true
	}
	if db == nil || !UserHasTheme(db, dbType, userID, code) {
		return themes.Theme{}, false
	}
	return ThemeByCode(db, dbType, code)
}

// ThemeByCode finds a theme by code: built-in or installed (by any user).
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
