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
	options?: Record<string, Record<string, string>>;
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
