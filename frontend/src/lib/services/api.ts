import type {
	ApiResponse,
	ListResponse,
	ListOptions,
	TableRecord
} from '$types/api';
import { handleApiResponse, handleApiResponseVoid, handleApiResponseFull, handleApiResponseWithCaptions, type DataWithCaptions } from '$lib/utils/apiHelpers';

const API_BASE = '/api';

// Helper to build query string from filters
function buildQueryString(options?: ListOptions): string {
	if (!options) return '';

	const params = new URLSearchParams();

	if (options.page) params.append('page', options.page.toString());
	if (options.page_size) params.append('page_size', options.page_size.toString());
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
	): Promise<{ list: ListResponse<T>; options: Record<string, Record<string, string>>; lookups: Record<string, Record<string, string>> }> {
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

	async getRecordIDs(tableName: string, sortBy?: string): Promise<string[]> {
		const url = `${API_BASE}/tables/${tableName}/ids${sortBy ? '?sort_by=' + sortBy : ''}`;
		const response = await fetch(url);
		const data = await handleApiResponse<{ ids: string[] }>(response, `get ${tableName} IDs`);
		return data.ids;
	},

	async getRecord<T = TableRecord>(tableName: string, id: string): Promise<T> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/card/${id}`);
		return handleApiResponse<T>(response, `get ${tableName} ${id}`);
	},

	async getRecordWithCaptions<T = TableRecord>(tableName: string, id: string): Promise<DataWithCaptions<T>> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/card/${id}`);
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

	async getTableOptionsAndLookups(tableName: string): Promise<{ options: Record<string, Record<string, string>>; lookups: Record<string, Record<string, string>> }> {
		// Fast endpoint that returns both options and lookup metadata (no records)
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
		const response = await fetch(`${API_BASE}/tables/${tableName}/modify/${id}`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});
		return handleApiResponse<T>(response, `modify ${tableName} ${id}`);
	},

	async deleteRecord(tableName: string, id: string): Promise<void> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/delete/${id}`, {
			method: 'DELETE'
		});
		return handleApiResponseVoid(response, `delete ${tableName} ${id}`);
	},

	async validateField(
		tableName: string,
		fieldName: string,
		value: any
	): Promise<{ valid: boolean; error?: string }> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/validate`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ field: fieldName, value })
		});

		if (!response.ok) {
			throw new Error(`Failed to validate field: ${response.statusText}`);
		}

		const result: ApiResponse = await response.json();
		return {
			valid: result.success,
			error: result.error
		};
	},

	// Run codeunit
	async runCodeunit(codeunitId: number, params?: any): Promise<any> {
		const response = await fetch(`${API_BASE}/codeunits/${codeunitId}/run`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(params || {})
		});
		return handleApiResponse<any>(response, `run codeunit ${codeunitId}`);
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

	async listCompanies(): Promise<ApiResponse<string[]>> {
		const response = await fetch(`${API_BASE}/auth/companies`);
		return handleApiResponseFull<string[]>(response, 'list companies');
	},

	async createCompany(name: string): Promise<ApiResponse<{ name: string; message: string }>> {
		const response = await fetch(`${API_BASE}/auth/companies`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name })
		});
		return handleApiResponseFull<{ name: string; message: string }>(response, 'create company');
	}
};
