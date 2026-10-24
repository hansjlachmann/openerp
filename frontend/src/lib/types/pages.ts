// Page type definitions matching the Go backend

export interface PageDefinition {
	page: PageMetadata;
}

export interface PageMetadata {
	id: number;
	type: 'Card' | 'List' | 'Document' | 'Worksheet';
	name: string;
	source_table: string;
	caption: string;
	card_page_id?: number;
	list_page_id?: number; // For card pages, the associated list page
	modal_card?: boolean;
	editable?: boolean;
	enable_navigation?: boolean;
	// DelayedInsert: insert a new list row only when the user leaves the row
	delayed_insert?: boolean;
	copy_paste_allowed?: boolean; // list Ctrl+C copies the selected records, Ctrl+V pastes them
	auto_split_key?: boolean; // a new line between two lines gets the line number between them
	focus_field?: string; // Field to focus when page opens
	layout: Layout;
	actions?: Action[];
	// Primary key field names from the source table (sent by the backend), used to
	// identify records even when the PK is not shown on the page (e.g. setup tables).
	primary_key_fields?: string[];
	has_record_triggers?: boolean;
	setup_table?: boolean; // a single record with a blank key (setup): the card opens on it
	auto_key?: boolean; // the table's OnInsert assigns a blank key (a number series): a new record is saved without one
	has_new_record_triggers?: boolean; // OnNewRecord: new records get values from their neighbour // OnAfterGetRecord: the card asks for the field states of each record
	flow_fields?: string[]; // computed fields: not searchable on the server
	sortable_flow_fields?: string[]; // FlowFields lists can sort on (Sum/Count, by their value)
	data_caption_fields?: string[]; // cards: the fields whose values follow the caption in the title
	// FlowFilter fields of the source table (e.g. Date Filter): "Filter totals by" in the filter pane
	flow_filter_fields?: Array<{ name: string; kind: string; caption: string }>;
}

export interface Layout {
	sections?: Section[];
	repeater?: Repeater;
}

export interface Section {
	name: string;
	caption: string;
	fields: Field[];
	subpage?: Subpage; // instead of fields: a list page linked to the card's record
}

/** A list page shown in a card section, tied to the card's record */
export interface Subpage {
	page_id: number;
	link: Record<string, string>; // subpage field → card field (e.g. document_no: "no")
}

export interface Repeater {
	fields: Field[];
}

export interface Field {
	source: string;
	caption?: string;
	editable?: boolean;
	visible?: boolean;
	importance?: 'Promoted' | 'Standard' | 'Additional';
	style?: 'Strong' | 'Attention' | 'Favorable' | 'Unfavorable';
	table_relation?: string;
	width?: number;
	drilldown?: number;
	drilldown_filter_field?: string;
	drilldown_filter_value?: string;
	primary_key?: boolean;
	required?: boolean;
	masked?: boolean; // write-only secret (Masked): password input, value never shown
	decimal_places?: string; // decimals: "min:max" digits after the point (table YAML decimal_places)
	indent_field?: string; // lists: indent the cell by this field's value (levels)
	style_expr?: StyleExpr; // lists: a style when the record's field (does not) equal a value
	assist_edit?: string; // AssistEdit: run_object of the field's "…" function (table:Customer.AssistEdit)
	calculated?: boolean; // not a table field: the page's triggers fill it per record (read-only)
}

/** A conditional cell style: style when record[field] equals `equal` (or differs from `not_equal`) */
export interface StyleExpr {
	field: string;
	equal?: string | number | boolean;
	not_equal?: string | number | boolean;
	style: 'Strong' | 'Attention' | 'Favorable' | 'Unfavorable';
}

export interface Action {
	name: string;
	caption: string;
	shortcut?: string;
	promoted?: boolean;
	run_page?: number;
	run_page_filter_field?: string; // open run_page filtered: filter_field = this record's
	run_page_filter_value?: string; // filter_value field (e.g. the customer's ledger entries)
	run_page_filters?: { field: string; value?: string; const?: string }[]; // several filters (record fields or constants)
	run_object?: string; // "codeunit:455", "report:675", a function "table:Job_Queue_Entry.ShowLineNo"
	enabled?: boolean;
	items?: Action[]; // a menu button: its menu items (each an action of its own)
	image?: string; // the button's icon (actionIcons.ts names, e.g. "New", "Post"); a bullet by default
}

export interface MenuDefinition {
	menu: MenuGroup[];
}

export interface MenuGroup {
	name: string;
	icon?: string;
	items?: MenuItem[]; // For grouped menus
	page_id?: number; // For flat menus (direct link)
}

export interface MenuItem {
	name?: string;
	page_id?: number;
	run_object?: string; // runs a report or codeunit ("report:93", "codeunit:6")
	items?: MenuItem[]; // a sub-menu (Setup, Activities …)
	icon?: string;
	description?: string;
	separator?: boolean;
	enabled?: boolean; // false: its object is not built yet (shown greyed out)
}

export interface PageResponse {
	success: boolean;
	data: PageDefinition;
	captions?: {
		table_name: string;
		fields: Record<string, string>;
	};
	navigation?: Record<string, string>; // Translated navigation strings (e.g., "home")
}

export interface MenuResponse {
	success: boolean;
	data: MenuDefinition;
}

/** Fields a page trigger switched for the current record (OnAfterGetRecord) */
export interface FieldState {
	editable?: boolean;
	visible?: boolean;
}

/** A data item of a report's request page: its fields can be filtered as on a list page */
export interface RequestDataItem {
	name: string;
	caption: string;
	indent: number; // 0 top level, 1 under its parent, …
	fields: RequestField[];
	filters: RequestFilter[]; // to start with (the request filter fields)
}

export interface RequestField {
	name: string;
	caption: string;
	type?: string;
}

export interface RequestFilter {
	field: string;
	expression: string;
}
