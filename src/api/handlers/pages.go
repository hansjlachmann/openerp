package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"

	apitypes "github.com/hansjlachmann/openerp/src/api/types"
	"github.com/hansjlachmann/openerp/src/foundation/i18n"
	"github.com/hansjlachmann/openerp/src/foundation/pages"
	"github.com/hansjlachmann/openerp/src/foundation/session"
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
		return c.Status(fiber.StatusBadRequest).JSON(apitypes.NewErrorResponse("Invalid page ID"))
	}

	// Get page definition from registry
	registry := pages.GetRegistry()
	pageDef, err := registry.GetPage(pageID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(apitypes.NewErrorResponse(fmt.Sprintf("Page %d not found", pageID)))
	}

	// Get current session for captions
	sess := session.GetCurrent()
	if sess == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(apitypes.NewErrorResponse("No active session"))
	}

	// Get field captions using i18n
	var captions *apitypes.CaptionData
	if pageDef.Page.SourceTable != "" {
		ts := i18n.GetInstance()
		lang := sess.GetLanguage()

		// Build field captions map
		fieldCaptions := make(map[string]string)

		// Get captions for card page sections
		for _, section := range pageDef.Page.Layout.Sections {
			for _, field := range section.Fields {
				fieldCaptions[field.Source] = ts.FieldCaption(pageDef.Page.SourceTable, field.Source, lang)
			}
		}

		// Get captions for list page repeater
		if pageDef.Page.Layout.Repeater != nil {
			for _, field := range pageDef.Page.Layout.Repeater.Fields {
				fieldCaptions[field.Source] = ts.FieldCaption(pageDef.Page.SourceTable, field.Source, lang)
			}
		}

		captions = &apitypes.CaptionData{
			Table:  ts.TableCaption(pageDef.Page.SourceTable, lang),
			Fields: fieldCaptions,
		}
	}

	return c.JSON(apitypes.APIResponse{
		Success:  true,
		Data:     pageDef,
		Captions: captions,
	})
}

// GetMenu returns the menu structure
// GET /api/menu
func (h *PagesHandler) GetMenu(c *fiber.Ctx) error {
	// Get menu from registry
	registry := pages.GetRegistry()
	menuDef := registry.GetMenu()
	if menuDef == nil {
		return c.Status(fiber.StatusNotFound).JSON(apitypes.NewErrorResponse("Menu not found"))
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
