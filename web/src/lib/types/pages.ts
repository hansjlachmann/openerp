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
	editable?: boolean;
	layout: Layout;
	actions?: Action[];
}

export interface Layout {
	sections?: Section[];
	repeater?: Repeater;
}

export interface Section {
	name: string;
	caption: string;
	fields: Field[];
}

export interface Repeater {
	fields: Field[];
}

export interface Field {
	source: string;
	caption?: string;
	editable?: boolean;
	importance?: 'Promoted' | 'Standard' | 'Additional';
	style?: 'Strong' | 'Attention' | 'Favorable' | 'Unfavorable';
	table_relation?: string;
	width?: number;
}

export interface Action {
	name: string;
	caption: string;
	shortcut?: string;
	promoted?: boolean;
	run_page?: number;
	run_object?: string;
	enabled?: boolean;
}

export interface MenuDefinition {
	menu: MenuGroup[];
}

export interface MenuGroup {
	name: string;
	icon?: string;
	items: MenuItem[];
}

export interface MenuItem {
	name?: string;
	page_id?: number;
	icon?: string;
	description?: string;
	separator?: boolean;
	enabled?: boolean;
}

export interface PageResponse {
	success: boolean;
	data: PageDefinition;
	captions?: {
		table_name: string;
		fields: Record<string, string>;
	};
}

export interface MenuResponse {
	success: boolean;
	data: MenuDefinition;
}
