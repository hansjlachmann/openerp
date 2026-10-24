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
	// DelayedInsert (DelayedInsert): on editable lists, insert a new record only when
	// the user leaves the row, not on the first validated field. For pages where the user
	// types a composite key (e.g. User Members), so no half-entered key is ever saved.
	DelayedInsert *bool `yaml:"delayed_insert,omitempty" json:"delayed_insert,omitempty"`
	// CopyPasteAllowed (CopyPasteAllowed): on list pages, Ctrl+C copies the selected
	// records and Ctrl+V inserts them (same table; new keys from the list's filters, or
	// another company)
	CopyPasteAllowed *bool `yaml:"copy_paste_allowed,omitempty" json:"copy_paste_allowed,omitempty"`
	// AutoSplitKey (AutoSplitKey): on list pages whose last primary key field is a line number
	// (int), a new line inserted between two lines gets the number halfway between them (lines
	// are numbered in steps of 10000) — document lines
	AutoSplitKey *bool    `yaml:"auto_split_key,omitempty" json:"auto_split_key,omitempty"`
	FocusField   string   `yaml:"focus_field,omitempty" json:"focus_field,omitempty"` // Field to focus when page opens
	Layout       Layout   `yaml:"layout" json:"layout"`
	Actions      []Action `yaml:"actions,omitempty" json:"actions,omitempty"`
	// DataCaptionFields (populated at request time from the table's data_caption_fields): the
	// fields whose values a card shows in its title
	DataCaptionFields []string `yaml:"-" json:"data_caption_fields,omitempty"`
	// HasRecordTriggers (populated at request time): the page has OnAfterGetRecord triggers
	// (business-logic/pages) — the card asks for the field states of each record it shows
	HasRecordTriggers bool `yaml:"-" json:"has_record_triggers,omitempty"`
	// HasNewRecordTriggers (populated at request time): the page has OnNewRecord triggers — a
	// new record's initial values come from /init with the page and the neighbouring record
	HasNewRecordTriggers bool `yaml:"-" json:"has_new_record_triggers,omitempty"`
	// SetupTable (populated at request time): the source table holds a single record with a
	// blank key (setup); its card page opens on that record
	SetupTable bool `yaml:"-" json:"setup_table,omitempty"`
	// AutoKey (populated at request time): the source table's OnInsert assigns a blank key (the
	// next number of a series) — a new record is saved without one
	AutoKey bool `yaml:"-" json:"auto_key,omitempty"`
	// PrimaryKeyFields is populated at request time from table metadata so the
	// frontend knows the record key even when the PK is not shown (e.g. setup tables).
	PrimaryKeyFields []string `yaml:"-" json:"primary_key_fields,omitempty"`
	// FlowFields (populated at request time) are computed, not stored: list pages can
	// not sort or search on them server-side.
	FlowFields []string `yaml:"-" json:"flow_fields,omitempty"`
	// SortableFlowFields (populated at request time): the FlowFields lists can sort on
	// (Sum/Count, sorted on the server by their calculated value)
	SortableFlowFields []string `yaml:"-" json:"sortable_flow_fields,omitempty"`
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
	// Subpage: instead of fields, the section shows a list page with the records linked to
	// the card's record (e.g. a document's lines)
	Subpage *Subpage `yaml:"subpage,omitempty" json:"subpage,omitempty"`
}

// Subpage is a list page shown in a card, tied to the card's record by Link (SubPageLink)
type Subpage struct {
	PageID int `yaml:"page_id" json:"page_id"` // the list page
	// Link: field of the subpage's table → field of the card's table (e.g. document_no: no);
	// the subpage shows only the records with these values (filter group 2, the user cannot
	// change it), and new records get them
	Link map[string]string `yaml:"link" json:"link"`
}

// Repeater represents a repeater (table) on a List page
type Repeater struct {
	Fields []Field `yaml:"fields" json:"fields"`
}

// Field represents a single field on a page
type Field struct {
	Source               string `yaml:"source" json:"source"`                                                     // Field name from table
	Caption              string `yaml:"caption,omitempty" json:"caption,omitempty"`                               // Override caption
	Editable             *bool  `yaml:"editable,omitempty" json:"editable,omitempty"`                             // false: read-only (also from the table field's editable: false)
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
	// Masked (ExtendedDatatype Masked): shown as a password input, never in clear text.
	// Set in the page YAML (e.g. the User card's virtual password field) or taken from the
	// table field's masked: true
	Masked bool `yaml:"masked,omitempty" json:"masked,omitempty"`
	// DecimalPlaces: the "min:max" digits after the point the page shows — the page field's
	// own (decimal_places in the page YAML; a calculated field's), else the table field's
	// (filled in at request time)
	DecimalPlaces string `yaml:"decimal_places,omitempty" json:"decimal_places,omitempty"`
	// IndentField (lists): indent the cell by this field's value of the record, one step per
	// level (the chart of accounts indents names by Indentation)
	IndentField string `yaml:"indent_field,omitempty" json:"indent_field,omitempty"`
	// StyleExpr (lists): the cell takes Style when the record's field equals / does not equal a
	// value (an option by its number) — bold heading and total accounts
	StyleExpr *StyleExpr `yaml:"style_expr,omitempty" json:"style_expr,omitempty"`
	// Calculated: not a table field — the page's OnAfterGetRecord trigger fills it per record
	// (page.SetValue; classic page variables shown in a column); read-only, not searchable or
	// sortable. Type (e.g. types.Date) formats it; its caption is tables.<table>.fields.<source>.
	Calculated bool `yaml:"calculated,omitempty" json:"calculated,omitempty"`
	// AssistEdit: the field's "…" (classic OnAssistEdit) runs this registered function
	// (run_object "table:Customer.AssistEdit", registered WithAssistEdit) with the card's record
	AssistEdit string `yaml:"assist_edit,omitempty" json:"assist_edit,omitempty"`
	Type       string `yaml:"type,omitempty" json:"-"`
}

// StyleExpr is a conditional style of a list cell: Style when Field = Equal (or <> NotEqual)
type StyleExpr struct {
	Field    string      `yaml:"field" json:"field"`
	Equal    interface{} `yaml:"equal,omitempty" json:"equal,omitempty"`
	NotEqual interface{} `yaml:"not_equal,omitempty" json:"not_equal,omitempty"`
	Style    string      `yaml:"style" json:"style"` // Strong, Attention, Favorable, Unfavorable
}

// RunPageFilter is one filter of an action's run_page: Field of the opened page = this record's
// Value field, or Const
type RunPageFilter struct {
	Field string `yaml:"field" json:"field"`
	Value string `yaml:"value,omitempty" json:"value,omitempty"`
	Const string `yaml:"const,omitempty" json:"const,omitempty"`
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
	// RunPageFilters (optional): several filters of run_page — each a field of the opened page
	// set to this record's field (value) or a constant (const, an option by its number);
	// classic RunFormLink=A=FIELD(x),B=CONST(y)
	RunPageFilters []RunPageFilter `yaml:"run_page_filters,omitempty" json:"run_page_filters,omitempty"`
	RunObject      string          `yaml:"run_object,omitempty" json:"run_object,omitempty"` // Run codeunit, report, etc.
	// Enabled: actions are enabled unless the YAML says enabled: false (a missing value is
	// omitted from the JSON; a plain bool sent false and disabled such actions and their shortcuts)
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
	// Items: the action is a menu button (a dropdown) with these menu items; each item is an
	// action of its own (run_object, run_page, shortcut). run_object can call a registered
	// function with the current record: "table:Job_Queue_Entry.ShowLineNo" (foundation/functions)
	Items []Action `yaml:"items,omitempty" json:"items,omitempty"`
	// Image: the button's icon — a name of the frontend's action icon set
	// (frontend/src/lib/components/icons/actionIcons.ts: New, Edit, Delete, Post, Print, …);
	// without one the button shows a bullet point
	Image string `yaml:"image,omitempty" json:"image,omitempty"`
}

// MenuDefinition represents the menu structure
type MenuDefinition struct {
	Menu []MenuGroup `yaml:"menu" json:"menu"`
}

// MenuGroup is a module of the main menu (General Ledger, Sales & Receivables …): its menu
// entries, or a direct page
type MenuGroup struct {
	Name   string     `yaml:"name" json:"name"`
	Icon   string     `yaml:"icon,omitempty" json:"icon,omitempty"`
	Items  []MenuItem `yaml:"items,omitempty" json:"items,omitempty"`     // the module's menu
	PageID int        `yaml:"page_id,omitempty" json:"page_id,omitempty"` // For flat menus (direct item)
}

// MenuItem is an entry of a module's menu: it opens a page (PageID), runs a report or codeunit
// (RunObject "report:93", "codeunit:6"), holds a sub-menu (Items, e.g. Setup, Activities) or is
// a separator. Enabled is set when the menu is sent: the entry's object exists (a sub-menu:
// one of its entries is available); an entry without one is shown greyed out (not built yet).
type MenuItem struct {
	Name        string     `yaml:"name,omitempty" json:"name,omitempty"`
	PageID      int        `yaml:"page_id,omitempty" json:"page_id,omitempty"`
	RunObject   string     `yaml:"run_object,omitempty" json:"run_object,omitempty"`
	Items       []MenuItem `yaml:"items,omitempty" json:"items,omitempty"`
	Icon        string     `yaml:"icon,omitempty" json:"icon,omitempty"`
	Description string     `yaml:"description,omitempty" json:"description,omitempty"`
	Separator   bool       `yaml:"separator,omitempty" json:"separator,omitempty"`
	Enabled     bool       `yaml:"-" json:"enabled"`
}

// cloneActions copies actions and their menu items (captions are translated per request)
func cloneActions(actions []Action) []Action {
	if actions == nil {
		return nil
	}
	out := make([]Action, len(actions))
	for i, a := range actions {
		a.Items = cloneActions(a.Items)
		out[i] = a
	}
	return out
}

// Clone returns a copy of the page definition that can be changed per request (translated
// captions, primary key and required flags) without touching the registry's shared one.
func (p *PageDefinition) Clone() *PageDefinition {
	c := *p
	m := &c.Page
	m.Layout.Sections = make([]Section, len(p.Page.Layout.Sections))
	for i, s := range p.Page.Layout.Sections {
		s.Fields = append([]Field{}, s.Fields...) // never null (a subpage section has none)
		m.Layout.Sections[i] = s
	}
	if p.Page.Layout.Repeater != nil {
		r := *p.Page.Layout.Repeater
		r.Fields = append([]Field(nil), r.Fields...)
		m.Layout.Repeater = &r
	}
	m.Actions = cloneActions(p.Page.Actions)
	m.PrimaryKeyFields = append([]string(nil), p.Page.PrimaryKeyFields...)
	m.FlowFields = append([]string(nil), p.Page.FlowFields...)
	m.SortableFlowFields = append([]string(nil), p.Page.SortableFlowFields...)
	m.FlowFilterFields = append([]FlowFilterField(nil), p.Page.FlowFilterFields...)
	return &c
}
