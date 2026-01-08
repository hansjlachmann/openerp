package pages

// PageDefinition represents a complete page definition from YAML
type PageDefinition struct {
	Page PageMetadata `yaml:"page" json:"page"`
}

// PageMetadata contains page metadata and structure
type PageMetadata struct {
	ID           int      `yaml:"id" json:"id"`
	Type         string   `yaml:"type" json:"type"` // Card, List, Document, Worksheet
	Name         string   `yaml:"name" json:"name"`
	SourceTable  string   `yaml:"source_table" json:"source_table"`
	Caption      string   `yaml:"caption" json:"caption"`
	CardPageID   int      `yaml:"card_page_id,omitempty" json:"card_page_id,omitempty"`
	ModalCard    bool     `yaml:"modal_card,omitempty" json:"modal_card,omitempty"`
	Editable     bool     `yaml:"editable,omitempty" json:"editable,omitempty"`
	Layout       Layout   `yaml:"layout" json:"layout"`
	Actions      []Action `yaml:"actions,omitempty" json:"actions,omitempty"`
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
	Source        string `yaml:"source" json:"source"`                                   // Field name from table
	Caption       string `yaml:"caption,omitempty" json:"caption,omitempty"`             // Override caption
	Editable      bool   `yaml:"editable,omitempty" json:"editable,omitempty"`           // Can be edited
	Importance    string `yaml:"importance,omitempty" json:"importance,omitempty"`       // Promoted, Standard, Additional
	Style         string `yaml:"style,omitempty" json:"style,omitempty"`                 // Strong, Attention, Favorable, Unfavorable
	TableRelation string `yaml:"table_relation,omitempty" json:"table_relation,omitempty"` // Lookup table
	Width         int    `yaml:"width,omitempty" json:"width,omitempty"`                 // Column width (for List pages)
}

// Action represents a page action/button
type Action struct {
	Name      string `yaml:"name" json:"name"`
	Caption   string `yaml:"caption" json:"caption"`
	Shortcut  string `yaml:"shortcut,omitempty" json:"shortcut,omitempty"`
	Promoted  bool   `yaml:"promoted,omitempty" json:"promoted,omitempty"`
	RunPage   int    `yaml:"run_page,omitempty" json:"run_page,omitempty"`       // Open another page
	RunObject string `yaml:"run_object,omitempty" json:"run_object,omitempty"`   // Run codeunit, report, etc.
	Enabled   bool   `yaml:"enabled,omitempty" json:"enabled"`                   // Default true
}

// MenuDefinition represents the menu structure
type MenuDefinition struct {
	Menu []MenuGroup `yaml:"menu" json:"menu"`
}

// MenuGroup represents a top-level menu group
type MenuGroup struct {
	Name  string     `yaml:"name" json:"name"`
	Icon  string     `yaml:"icon,omitempty" json:"icon,omitempty"`
	Items []MenuItem `yaml:"items" json:"items"`
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
