import type {
	ApiResponse,
	ListResponse,
	ListOptions,
	TableRecord,
	LookupData,
	CodeunitResult,
	ValidateFieldResult,
	TableFilter
} from '$types/api';
import { handleApiResponse, handleApiResponseVoid, handleApiResponseFull, handleApiResponseWithCaptions, type DataWithCaptions } from '$lib/utils/apiHelpers';
import type { CompanyInfo } from '$lib/utils/company';

const API_BASE = '/api';

// Builds a record endpoint URL. An empty id (a BC-style setup table's single
// record has a blank primary key) omits the id segment so it matches the no-id
// backend routes instead of producing a trailing-slash 404.
function recordUrl(tableName: string, action: 'card' | 'modify' | 'delete', id: string): string {
	return id === ''
		? `${API_BASE}/tables/${tableName}/${action}`
		: `${API_BASE}/tables/${tableName}/${action}/${id}`;
}

// Helper to build query string from filters
function buildQueryString(options?: ListOptions): string {
	if (!options) return '';

	const params = new URLSearchParams();

	if (options.page) params.append('page', options.page.toString());
	if (options.page_size) params.append('page_size', options.page_size.toString());
	if (options.limit) {
		params.append('offset', (options.offset ?? 0).toString());
		params.append('limit', options.limit.toString());
	}
	if (options.flow_filters && options.flow_filters.length > 0) {
		params.append('flow_filters', JSON.stringify(options.flow_filters));
	}
	if (options.search && options.search_fields && options.search_fields.length > 0) {
		params.append('search', options.search);
		params.append('search_fields', JSON.stringify(options.search_fields));
	}
	if (options.sort_by) params.append('sort_by', options.sort_by);
	if (options.sort_order) params.append('sort_order', options.sort_order);

	// Add filters as JSON
	if (options.filters && options.filters.length > 0) {
		params.append('filters', JSON.stringify(options.filters));
	}

	// Add fields as JSON
	if (options.fields && options.fields.length > 0) {
		params.append('fields', JSON.stringify(options.fields));
	}

	return params.toString();
}

// Generic API client
export const api = {
	// Generic table operations
	async listRecords<T = TableRecord>(
		tableName: string,
		options?: ListOptions
	): Promise<ListResponse<T>> {
		const query = buildQueryString(options);
		const url = `${API_BASE}/tables/${tableName}/list${query ? '?' + query : ''}`;
		const response = await fetch(url);
		return handleApiResponse<ListResponse<T>>(response, `list ${tableName}`);
	},

	async listRecordsWithOptions<T = TableRecord>(
		tableName: string,
		listOptions?: ListOptions
	): Promise<{ list: ListResponse<T>; options: Record<string, Record<string, string>>; lookups: Record<string, LookupData> }> {
		const query = buildQueryString(listOptions);
		const url = `${API_BASE}/tables/${tableName}/list${query ? '?' + query : ''}`;
		const response = await fetch(url);
		const result = await handleApiResponseWithCaptions<ListResponse<T>>(response, `list ${tableName}`);
		return {
			list: result.data,
			options: result.captions?.options || {},
			lookups: result.captions?.lookups || {}
		};
	},

	// Rows for an on-demand dropdown (LookupData.lazy_url): matching search, or the row with key
	async lookupRows(
		lazyUrl: string,
		params: { search?: string; key?: string }
	): Promise<{ rows: Array<{ _key: string; [key: string]: any }>; total: number }> {
		const query = new URLSearchParams();
		if (params.search) query.append('search', params.search);
		if (params.key) query.append('key', params.key);
		const response = await fetch(`${lazyUrl}${query.toString() ? '?' + query.toString() : ''}`);
		return handleApiResponse(response, 'load lookup rows');
	},

	async getRecordIDs(tableName: string, sortBy?: string): Promise<string[]> {
		const url = `${API_BASE}/tables/${tableName}/ids${sortBy ? '?sort_by=' + sortBy : ''}`;
		const response = await fetch(url);
		const data = await handleApiResponse<{ ids: string[] }>(response, `get ${tableName} IDs`);
		return data.ids;
	},

	async getRecord<T = TableRecord>(tableName: string, id: string): Promise<T> {
		const response = await fetch(recordUrl(tableName, 'card', id));
		return handleApiResponse<T>(response, `get ${tableName} ${id}`);
	},

	async getRecordWithCaptions<T = TableRecord>(tableName: string, id: string, flowFilters?: TableFilter[]): Promise<DataWithCaptions<T>> {
		const query = flowFilters && flowFilters.length > 0 ? `?flow_filters=${encodeURIComponent(JSON.stringify(flowFilters))}` : '';
		const response = await fetch(recordUrl(tableName, 'card', id) + query);
		return handleApiResponseWithCaptions<T>(response, `get ${tableName} ${id}`);
	},

	async getTableOptions(tableName: string): Promise<Record<string, Record<string, string>>> {
		// Fast endpoint that only returns option metadata (no records)
		if (!tableName) {
			console.warn('getTableOptions called with empty tableName');
			return {};
		}
		const response = await fetch(`${API_BASE}/tables/${tableName}/options`);
		if (!response.ok) {
			console.error(`getTableOptions failed for ${tableName}: ${response.statusText}`);
			return {};
		}
		const result = await response.json();
		return result.data?.options || {};
	},

	async getTableOptionsAndLookups(tableName: string): Promise<{ options: Record<string, Record<string, string>>; lookups: Record<string, LookupData> }> {
		// Fast endpoint that returns both options and lookup data (includes columns, rows for advanced lookups)
		if (!tableName) {
			console.warn('getTableOptionsAndLookups called with empty tableName');
			return { options: {}, lookups: {} };
		}
		const response = await fetch(`${API_BASE}/tables/${tableName}/options`);
		if (!response.ok) {
			console.error(`getTableOptionsAndLookups failed for ${tableName}: ${response.statusText}`);
			return { options: {}, lookups: {} };
		}
		const result = await response.json();
		return {
			options: result.data?.options || {},
			lookups: result.data?.lookups || {}
		};
	},

	async insertRecord<T = TableRecord>(tableName: string, data: Partial<T>): Promise<T> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/insert`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});
		return handleApiResponse<T>(response, `insert ${tableName}`);
	},

	async modifyRecord<T = TableRecord>(
		tableName: string,
		id: string,
		data: Partial<T>
	): Promise<T> {
		const response = await fetch(recordUrl(tableName, 'modify', id), {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});
		return handleApiResponse<T>(response, `modify ${tableName} ${id}`);
	},

	async deleteRecord(tableName: string, id: string): Promise<void> {
		const response = await fetch(recordUrl(tableName, 'delete', id), {
			method: 'DELETE'
		});
		return handleApiResponseVoid(response, `delete ${tableName} ${id}`);
	},

	// Validate a field (BC/NAV VALIDATE). Pass the in-progress record so the field's
	// OnValidate trigger can see and fill in sibling fields; the result carries the
	// resulting record.
	async validateField(
		tableName: string,
		fieldName: string,
		value: any,
		record?: TableRecord
	): Promise<ValidateFieldResult> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/validate`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ field: fieldName, value, record })
		});

		if (!response.ok) {
			throw new Error(`Failed to validate field: ${response.statusText}`);
		}

		const result: ApiResponse<TableRecord> = await response.json();
		return {
			valid: result.success,
			error: result.error,
			record: result.data
		};
	},

	// Get a new, not yet inserted record with its defaults applied (BC/NAV OnNewRecord)
	async initRecord<T = TableRecord>(tableName: string): Promise<T> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/init`, {
			method: 'POST'
		});
		return handleApiResponse<T>(response, `init ${tableName}`);
	},

	// Run codeunit by ID with record data
	async runCodeunit(codeunitId: number, record: Record<string, any>): Promise<CodeunitResult> {
		const response = await fetch(`${API_BASE}/codeunits/run`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ codeunit_id: codeunitId, record })
		});
		return handleApiResponse<CodeunitResult>(response, `run codeunit ${codeunitId}`);
	},

	// Authentication
	async login(userID: string, password: string, company?: string): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/login`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ user_id: userID, password, company })
		});
		return handleApiResponseFull(response, 'login');
	},

	async logout(): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/logout`, {
			method: 'POST'
		});
		return handleApiResponseFull(response, 'logout');
	},

	async getCurrentUser(): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/user`);
		return handleApiResponseFull(response, 'get current user');
	},

	async createInitialUser(data: {
		user_id: string;
		user_name: string;
		email: string;
		password: string;
	}): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/init`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});
		return handleApiResponseFull(response, 'create initial user');
	},

	async setLanguage(language: string, persist: boolean = true): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/language`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ language, persist })
		});
		return handleApiResponseFull(response, 'set language');
	},

	async getLanguages(): Promise<{ code: string; name: string }[]> {
		const response = await fetch(`${API_BASE}/auth/languages`);
		const result = await handleApiResponseFull<{ code: string; name: string }[]>(response, 'get languages');
		return result.data || [];
	},

	async listCompanies(): Promise<ApiResponse<CompanyInfo[]>> {
		const response = await fetch(`${API_BASE}/auth/companies`);
		return handleApiResponseFull<CompanyInfo[]>(response, 'list companies');
	},

	async getCompanies(): Promise<CompanyInfo[]> {
		const response = await fetch(`${API_BASE}/auth/companies`);
		const result = await handleApiResponseFull<CompanyInfo[]>(response, 'get companies');
		return result.data || [];
	},

	async setCompany(company: string): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/company`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ company })
		});
		return handleApiResponseFull(response, 'set company');
	},

	async createCompany(name: string): Promise<ApiResponse<{ name: string; message: string }>> {
		const response = await fetch(`${API_BASE}/auth/companies`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name })
		});
		return handleApiResponseFull<{ name: string; message: string }>(response, 'create company');
	},

	// Version
	async getVersion(): Promise<string> {
		try {
			const response = await fetch(`${API_BASE}/version`);
			if (!response.ok) return 'unknown';
			const result = await response.json();
			return result.version || 'unknown';
		} catch {
			return 'unknown';
		}
	}
};
