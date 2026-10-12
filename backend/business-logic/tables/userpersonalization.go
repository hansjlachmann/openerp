package tables

import (
	"github.com/hansjlachmann/openerp/backend/foundation/database"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	ftables "github.com/hansjlachmann/openerp/backend/foundation/tables"
	"github.com/hansjlachmann/openerp/backend/foundation/themes"
	"github.com/hansjlachmann/openerp/backend/foundation/types"
	gtables "github.com/hansjlachmann/openerp/backend/generated/tables"
)

//go:generate go run ../../../tools/tablegen/main.go

// UserPersonalization (BC User Personalization) holds a user's appearance: the theme for
// light mode and the one for dark mode, the current mode, wallpaper, font and density.
// One record per user, global (the same in every company); a user without a record gets
// DefaultUserPersonalization.
type UserPersonalization struct {
	gtables.UserPersonalizationBase
}

// Limits of the appearance settings
const (
	WallpaperOpacityMax = 100
	WallpaperBlurMax    = 20
	PanelOpacityMin     = 75 // panels stay opaque enough to read lists on any wallpaper
	FontSizeMin         = 80 // percent of the default; 0 = default
	FontSizeMax         = 150
)

// DefaultUserPersonalization returns the appearance of a user without a record: Business
// Central Light/Dark, dark mode (as before themes existed), no wallpaper.
func DefaultUserPersonalization(userID string) *UserPersonalization {
	p := &UserPersonalization{}
	p.User_id = types.NewCode(userID)
	p.Light_theme = types.NewCode(themes.DefaultLight)
	p.Dark_theme = types.NewCode(themes.DefaultDark)
	p.Mode = gtables.UserPersonalization_Mode.Dark
	p.Wallpaper_opacity = 25
	p.Panel_opacity = 92
	return p
}

// ThemeByCode finds a theme the user can choose.
func ThemeByCode(code string) (themes.Theme, bool) {
	return themes.Builtin(code)
}

// NewUserPersonalization creates a new UserPersonalization instance
func NewUserPersonalization() *UserPersonalization {
	return &UserPersonalization{}
}

// InitWithDBType initializes the record with database context and type and sets up
// triggers. The API creates tables via the tables.Table interface and calls this method,
// so the wiring must live here for triggers and OnValidate_* overrides to fire.
func (t *UserPersonalization) InitWithDBType(db database.Executor, company string, dbType database.DBType) {
	t.UserPersonalizationBase.InitWithDBType(db, company, dbType)
	t.SetTriggers(t.OnInsert, t.OnModify, t.OnDelete)
	t.SetSelf(t)
}

// ========================================
// Table Triggers (Business Logic)
// ========================================

// OnInsert trigger - called before inserting a new record
func (t *UserPersonalization) OnInsert() error {
	return t.Validate()
}

// OnModify trigger - called before modifying a record
func (t *UserPersonalization) OnModify() error {
	return t.Validate()
}

// OnDelete trigger - called before deleting a record
func (t *UserPersonalization) OnDelete(db database.Executor, company string) error {
	// TODO: Add checks for related records (if any)
	return nil
}

// OnRename trigger - called before renaming (changing primary key)
func (t *UserPersonalization) OnRename() error {
	// TODO: Update related records if needed
	return nil
}

// ========================================
// Validation
// ========================================

// Validate validates all fields
func (t *UserPersonalization) Validate() error {
	if err := ftables.CheckRequired(gtables.UserPersonalizationTableName, "user_id", t.User_id.IsEmpty()); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.UserPersonalizationTableName, "user_id", string(t.User_id), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.UserPersonalizationTableName, "light_theme", string(t.Light_theme), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.UserPersonalizationTableName, "dark_theme", string(t.Dark_theme), 50); err != nil {
		return err
	}
	if err := ftables.CheckMaxLength(gtables.UserPersonalizationTableName, "wallpaper", string(t.Wallpaper), 20); err != nil {
		return err
	}
	if err := t.OnValidate_Light_theme(); err != nil {
		return err
	}
	if err := t.OnValidate_Dark_theme(); err != nil {
		return err
	}
	table := gtables.UserPersonalizationTableName
	if err := ftables.CheckRange(table, "wallpaper_opacity", t.Wallpaper_opacity, 0, WallpaperOpacityMax); err != nil {
		return err
	}
	if err := ftables.CheckRange(table, "wallpaper_blur", t.Wallpaper_blur, 0, WallpaperBlurMax); err != nil {
		return err
	}
	if err := ftables.CheckRange(table, "panel_opacity", t.Panel_opacity, PanelOpacityMin, 100); err != nil {
		return err
	}
	if t.Font_size != 0 {
		if err := ftables.CheckRange(table, "font_size", t.Font_size, FontSizeMin, FontSizeMax); err != nil {
			return err
		}
	}
	if !t.Mode.IsValid() {
		return apperrors.FieldOutOfRange(table, "mode", "0", "1")
	}
	if !t.Font.IsValid() {
		return apperrors.FieldOutOfRange(table, "font", "0", "5")
	}
	if !t.Density.IsValid() {
		return apperrors.FieldOutOfRange(table, "density", "0", "1")
	}
	return nil
}

// OnValidate_Light_theme checks that the light theme exists and is made for light mode
func (t *UserPersonalization) OnValidate_Light_theme() error {
	return checkThemeSlot("light_theme", t.Light_theme.String(), "light")
}

// OnValidate_Dark_theme checks that the dark theme exists and is made for dark mode
func (t *UserPersonalization) OnValidate_Dark_theme() error {
	return checkThemeSlot("dark_theme", t.Dark_theme.String(), "dark")
}

func checkThemeSlot(field, code, mode string) error {
	table := gtables.UserPersonalizationTableName
	theme, ok := ThemeByCode(code)
	if !ok {
		return apperrors.RelatedNotFound(table, field, code)
	}
	if theme.Mode != mode {
		return apperrors.ThemeWrongMode(table, field, theme.Name)
	}
	return nil
}

// ========================================
// Business Logic Methods
// ========================================

// TODO: Add your custom business logic methods here
// Example:
// func (t *UserPersonalization) CalculateSomething() error {
//     // Your logic here
//     return nil
// }
