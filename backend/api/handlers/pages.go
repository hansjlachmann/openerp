package handlers

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	apitypes "github.com/hansjlachmann/openerp/backend/api/types"
	apperrors "github.com/hansjlachmann/openerp/backend/foundation/errors"
	"github.com/hansjlachmann/openerp/backend/foundation/i18n"
	"github.com/hansjlachmann/openerp/backend/foundation/pages"
	"github.com/hansjlachmann/openerp/backend/foundation/session"
)

// PagesHandler handles page-related API endpoints
type PagesHandler struct{}

// NewPagesHandler creates a new pages handler
func NewPagesHandler() *PagesHandler {
	return &PagesHandler{}
}

// GetPage returns a page definition by ID
// GET /api/pages/:id
func (h *PagesHandler) GetPage(c *fiber.Ctx) error {
	// Get page ID from URL parameter
	pageIDStr := c.Params("id")
	pageID, err := strconv.Atoi(pageIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apitypes.NewErrorResponse(apperrors.InvalidPageID().Message("en-US")))
	}

	// Get page definition from registry
	registry := pages.GetRegistry()
	pageDef, err := registry.GetPage(pageID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(apitypes.NewErrorResponse(apperrors.PageNotFound(pageIDStr).Message("en-US")))
	}

	// Get current session for captions
	sess := session.GetCurrent()
	if sess == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apitypes.NewErrorResponse(apperrors.NoActiveSession().Message("en-US")))
	}

	// Get table metadata for primary key info
	tableMeta := pages.GetTableMetadata()
	primaryKeyFields := tableMeta.GetPrimaryKeyFields(pageDef.Page.SourceTable)
	// Build a set for quick lookup
	pkSet := make(map[string]bool, len(primaryKeyFields))
	for _, pk := range primaryKeyFields {
		pkSet[pk] = true
	}

	// Get field captions using i18n
	ts := i18n.GetInstance()
	lang := normalizeLanguageCode(sess.GetLanguage())
	// Fallback to en-US if language is not set
	if lang == "" {
		lang = "en-US"
	}

	var captions *apitypes.CaptionData
	if pageDef.Page.SourceTable != "" {
		// Build field captions map
		fieldCaptions := make(map[string]string)

		// Get captions for card page sections and mark primary key fields
		for i := range pageDef.Page.Layout.Sections {
			for j := range pageDef.Page.Layout.Sections[i].Fields {
				field := &pageDef.Page.Layout.Sections[i].Fields[j]
				fieldCaptions[field.Source] = ts.FieldCaption(pageDef.Page.SourceTable, field.Source, lang)
				if pkSet[field.Source] {
					field.PrimaryKey = true
				}
			}
		}

		// Get captions for list page repeater and mark primary key fields
		if pageDef.Page.Layout.Repeater != nil {
			for i := range pageDef.Page.Layout.Repeater.Fields {
				field := &pageDef.Page.Layout.Repeater.Fields[i]
				fieldCaptions[field.Source] = ts.FieldCaption(pageDef.Page.SourceTable, field.Source, lang)
				if pkSet[field.Source] {
					field.PrimaryKey = true
				}
			}
		}

		captions = &apitypes.CaptionData{
			Table:  ts.TableCaption(pageDef.Page.SourceTable, lang),
			Fields: fieldCaptions,
		}
	}

	// Translate page caption
	translatedCaption := ts.PageCaption(pageDef.Page.Name, lang)
	// If no translation found (key returned), use original caption
	expectedKey := fmt.Sprintf("pages.%s.caption", normalizePageName(pageDef.Page.Name))
	if translatedCaption != expectedKey {
		pageDef.Page.Caption = translatedCaption
	}

	// Build navigation translations for breadcrumbs
	navigation := map[string]string{
		"home": ts.CommonTranslation("navigation.home", lang),
	}

	return c.JSON(apitypes.APIResponse{
		Success:    true,
		Data:       pageDef,
		Captions:   captions,
		Navigation: navigation,
	})
}

// normalizePageName converts "Customer Card" to "customer_card"
func normalizePageName(name string) string {
	result := strings.ToLower(name)
	result = strings.ReplaceAll(result, " ", "_")
	return result
}

// normalizeLanguageCode converts language codes to proper BCP-47 format
// e.g., "NB-NO" -> "nb-NO", "EN-US" -> "en-US"
func normalizeLanguageCode(code string) string {
	if code == "" {
		return ""
	}
	parts := strings.Split(code, "-")
	if len(parts) == 2 {
		return strings.ToLower(parts[0]) + "-" + strings.ToUpper(parts[1])
	}
	return strings.ToLower(code)
}

// GetMenu returns the menu structure based on user's assigned menu
// GET /api/menu
func (h *PagesHandler) GetMenu(c *fiber.Ctx) error {
	// Get current session to determine user's menu
	sess := session.GetCurrent()
	menuName := ""
	if sess != nil {
		menuName = sess.GetMenu()
	}

	// Default to admin if no menu assigned
	if menuName == "" {
		menuName = "admin"
	}

	// Get menu from registry by name
	registry := pages.GetRegistry()
	menuDef := registry.GetMenuByName(menuName)
	if menuDef == nil {
		return c.Status(fiber.StatusNotFound).JSON(apitypes.NewErrorResponse(apperrors.MenuNotFound().Message("en-US")))
	}

	return c.JSON(apitypes.NewSuccessResponse(menuDef))
}

// GetAllPages returns all loaded page definitions
// GET /api/pages
func (h *PagesHandler) GetAllPages(c *fiber.Ctx) error {
	registry := pages.GetRegistry()
	allPages := registry.GetAllPages()

	// Convert map to slice for easier consumption
	pageList := make([]*pages.PageDefinition, 0, len(allPages))
	for _, page := range allPages {
		pageList = append(pageList, page)
	}

	return c.JSON(apitypes.NewSuccessResponse(pageList))
}
