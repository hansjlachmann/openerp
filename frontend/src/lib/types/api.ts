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

// Result of a field validation (POST /tables/:table/validate)
export interface ValidateFieldResult {
	valid: boolean;
	error?: string;
	record?: TableRecord; // Record after the OnValidate trigger ran (may fill sibling fields)
}

// List response with pagination
export interface ListResponse<T = TableRecord> {
	records: T[];
	total: number;
	page: number;
	page_size: number;
	offset?: number; // position of the first record in the full list (windowed loads)
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
	offset?: number; // window: skip this many records ...
	limit?: number; // ... and return at most this many (total = all matching records)
	search?: string; // case-insensitive "contains" over search_fields
	search_fields?: string[];
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
