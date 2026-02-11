// API Response types
export interface ApiResponse<T = any> {
	success: boolean;
	data?: T;
	error?: string;
	captions?: CaptionData;
}

export interface CaptionData {
	table?: string;
	fields?: Record<string, string>;
	field_types?: Record<string, string>;
	options?: Record<string, Record<string, string>>;
	lookups?: Record<string, LookupData>;
}

// Lookup data structures (for table relations / dropdowns)
export interface LookupColumn {
	source: string;
	width: number;
}

export interface LookupData {
	columns?: LookupColumn[];
	rows?: Array<{ _key: string; [key: string]: any }>;
	simple?: Record<string, string>;
	search_timeout?: number;
}

// Table record type (generic)
export interface TableRecord {
	[key: string]: any;
}

// List response with pagination
export interface ListResponse<T = TableRecord> {
	records: T[];
	total: number;
	page: number;
	page_size: number;
}

// Filter types (BC/NAV style)
export interface TableFilter {
	field: string;
	expression: string; // BC-style filter expression: supports *, |, .., <, >, etc.
}

export interface ListOptions {
	filters?: TableFilter[];
	sort_by?: string;
	sort_order?: 'asc' | 'desc';
	page?: number;
	page_size?: number;
	fields?: string[]; // Only load these fields (useful to skip expensive FlowFields)
}

// Dialog result from codeunit
export interface DialogResult {
	title: string;
	message: string;
	type: 'info' | 'success' | 'warning' | 'error';
}

// Codeunit execution result
export interface CodeunitResult {
	success: boolean;
	message: string;
	data?: Record<string, any>;
	dialog?: DialogResult;
}
