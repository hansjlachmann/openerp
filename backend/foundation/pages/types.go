package pages

// PageDefinition represents a complete page definition from YAML
type PageDefinition struct {
	Page PageMetadata `yaml:"page" json:"page"`
}

// PageMetadata contains page metadata and structure
type PageMetadata struct {
	ID               int    `yaml:"id" json:"id"`
	Type             string `yaml:"type" json:"type"` // Card, List, Document, Worksheet
	Name             string `yaml:"name" json:"name"`
	SourceTable      string `yaml:"source_table" json:"source_table"`
	Caption          string `yaml:"caption" json:"caption"`
	CardPageID       int    `yaml:"card_page_id,omitempty" json:"card_page_id,omitempty"`
	ListPageID       int    `yaml:"list_page_id,omitempty" json:"list_page_id,omitempty"` // For card pages, the associated list page
	ModalCard        *bool  `yaml:"modal_card,omitempty" json:"modal_card,omitempty"`
	Editable         *bool  `yaml:"editable,omitempty" json:"editable,omitempty"`
	EnableNavigation *bool  `yaml:"enable_navigation,omitempty" json:"enable_navigation,omitempty"`
	// DelayedInsert (BC/NAV DelayedInsert): on editable lists, insert a new record only when
	// the user leaves the row, not on the first validated field. For pages where the user
	// types a composite key (e.g. User Members), so no half-entered key is ever saved.
	DelayedInsert *bool    `yaml:"delayed_insert,omitempty" json:"delayed_insert,omitempty"`
	FocusField    string   `yaml:"focus_field,omitempty" json:"focus_field,omitempty"` // Field to focus when page opens
	Layout        Layout   `yaml:"layout" json:"layout"`
	Actions       []Action `yaml:"actions,omitempty" json:"actions,omitempty"`
	// PrimaryKeyFields is populated at request time from table metadata so the
	// frontend knows the record key even when the PK is not shown (e.g. setup tables).
	PrimaryKeyFields []string `yaml:"-" json:"primary_key_fields,omitempty"`
	// FlowFields (populated at request time) are computed, not stored: list pages can
	// not sort or search on them server-side.
	FlowFields []string `yaml:"-" json:"flow_fields,omitempty"`
	// FlowFilterFields (populated at request time): the source table's FlowFilter fields
	// (e.g. Date Filter), offered in the filter pane under "Filter totals by"
	FlowFilterFields []FlowFilterField `yaml:"-" json:"flow_filter_fields,omitempty"`
}

// FlowFilterField is a FlowFilter field as the page sends it: name, kind and caption
type FlowFilterField struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"` // date, bool, int, text, code
	Caption string `json:"caption"`
}

// Layout defines the page layout structure
type Layout struct {
	Sections []Section `yaml:"sections,omitempty" json:"sections,omitempty"` // For Card pages
	Repeater *Repeater `yaml:"repeater,omitempty" json:"repeater,omitempty"` // For List pages
}

// Section represents a section/group on a Card page
type Section struct {
	Name    string  `yaml:"name" json:"name"`
	Caption string  `yaml:"caption" json:"caption"`
	Fields  []Field `yaml:"fields" json:"fields"`
}

// Repeater represents a repeater (table) on a List page
type Repeater struct {
	Fields []Field `yaml:"fields" json:"fields"`
}

// Field represents a single field on a page
type Field struct {
	Source               string `yaml:"source" json:"source"`                                                     // Field name from table
	Caption              string `yaml:"caption,omitempty" json:"caption,omitempty"`                               // Override caption
	Editable             bool   `yaml:"editable,omitempty" json:"editable,omitempty"`                             // Can be edited
	Visible              *bool  `yaml:"visible,omitempty" json:"visible,omitempty"`                               // Field is visible (default true)
	Importance           string `yaml:"importance,omitempty" json:"importance,omitempty"`                         // Promoted, Standard, Additional
	Style                string `yaml:"style,omitempty" json:"style,omitempty"`                                   // Strong, Attention, Favorable, Unfavorable
	TableRelation        string `yaml:"table_relation,omitempty" json:"table_relation,omitempty"`                 // Lookup table
	Width                int    `yaml:"width,omitempty" json:"width,omitempty"`                                   // Column width (for List pages)
	Drilldown            int    `yaml:"drilldown,omitempty" json:"drilldown,omitempty"`                           // Page ID to navigate to on click
	DrilldownFilterField string `yaml:"drilldown_filter_field,omitempty" json:"drilldown_filter_field,omitempty"` // Field on target table to filter
	DrilldownFilterValue string `yaml:"drilldown_filter_value,omitempty" json:"drilldown_filter_value,omitempty"` // Field on current record for filter value
	PrimaryKey           bool   `yaml:"-" json:"primary_key,omitempty"`                                           // Is this the primary key field (populated at runtime)
	Required             bool   `yaml:"-" json:"required,omitempty"`                                              // Is this field required (populated at runtime)
}

// Action represents a page action/button
type Action struct {
	Name     string `yaml:"name" json:"name"`
	Caption  string `yaml:"caption" json:"caption"`
	Shortcut string `yaml:"shortcut,omitempty" json:"shortcut,omitempty"`
	Promoted bool   `yaml:"promoted,omitempty" json:"promoted,omitempty"`
	RunPage  int    `yaml:"run_page,omitempty" json:"run_page,omitempty"` // Open another page
	// RunPageFilterField / RunPageFilterValue (optional): open run_page filtered to
	// filter_field = this record's filter_value field (e.g. the customer's ledger entries)
	RunPageFilterField string `yaml:"run_page_filter_field,omitempty" json:"run_page_filter_field,omitempty"`
	RunPageFilterValue string `yaml:"run_page_filter_value,omitempty" json:"run_page_filter_value,omitempty"`
	RunObject          string `yaml:"run_object,omitempty" json:"run_object,omitempty"` // Run codeunit, report, etc.
	// Enabled: actions are enabled unless the YAML says enabled: false (a missing value is
	// omitted from the JSON; a plain bool sent false and disabled such actions and their shortcuts)
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
}

// MenuDefinition represents the menu structure
type MenuDefinition struct {
	Menu []MenuGroup `yaml:"menu" json:"menu"`
}

// MenuGroup represents a top-level menu group or direct menu item
type MenuGroup struct {
	Name   string     `yaml:"name" json:"name"`
	Icon   string     `yaml:"icon,omitempty" json:"icon,omitempty"`
	Items  []MenuItem `yaml:"items,omitempty" json:"items,omitempty"`     // For grouped menus
	PageID int        `yaml:"page_id,omitempty" json:"page_id,omitempty"` // For flat menus (direct item)
}

// MenuItem represents a menu item
type MenuItem struct {
	Name        string `yaml:"name,omitempty" json:"name,omitempty"`
	PageID      int    `yaml:"page_id,omitempty" json:"page_id,omitempty"`
	Icon        string `yaml:"icon,omitempty" json:"icon,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Separator   bool   `yaml:"separator,omitempty" json:"separator,omitempty"`
	Enabled     bool   `yaml:"enabled,omitempty" json:"enabled"`
}
