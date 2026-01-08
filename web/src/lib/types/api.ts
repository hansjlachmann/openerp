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
	operator: 'eq' | 'ne' | 'gt' | 'gte' | 'lt' | 'lte' | 'like' | 'in' | 'between';
	value: any;
	value2?: any; // For BETWEEN operator
}

export interface ListOptions {
	filters?: TableFilter[];
	sort_by?: string;
	sort_order?: 'asc' | 'desc';
	page?: number;
	page_size?: number;
}

// Customer type (specific)
export interface Customer {
	no: string;
	name: string;
	address?: string;
	post_code?: string;
	city?: string;
	phone_number?: string;
	email?: string;
	payment_terms_code?: string;
	credit_limit?: string;
	balance_lcy?: string;
	sales_lcy?: string;
	no_of_ledger_entries?: number;
	last_order_date?: string;
	created_at?: string;
	status?: number;
}

// Payment Terms type (specific)
export interface PaymentTerms {
	code: string;
	description: string;
	due_date_calculation?: string;
	discount_date_calculation?: string;
	discount_percent?: string;
}
