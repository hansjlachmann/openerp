import type {
	ApiResponse,
	ListResponse,
	ListOptions,
	TableRecord,
	TableFilter
} from '$types/api';

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
		if (!response.ok) {
			throw new Error(`Failed to list ${tableName}: ${response.statusText}`);
		}

		const result: ApiResponse<ListResponse<T>> = await response.json();
		if (!result.success) {
			throw new Error(result.error || 'Unknown error');
		}

		return result.data!;
	},

	async getRecordIDs(tableName: string, sortBy?: string): Promise<string[]> {
		const url = `${API_BASE}/tables/${tableName}/ids${sortBy ? '?sort_by=' + sortBy : ''}`;

		const response = await fetch(url);
		if (!response.ok) {
			throw new Error(`Failed to get ${tableName} IDs: ${response.statusText}`);
		}

		const result: ApiResponse<{ ids: string[] }> = await response.json();
		if (!result.success) {
			throw new Error(result.error || 'Unknown error');
		}

		return result.data!.ids;
	},

	async getRecord<T = TableRecord>(tableName: string, id: string): Promise<T> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/card/${id}`);
		if (!response.ok) {
			throw new Error(`Failed to get ${tableName} ${id}: ${response.statusText}`);
		}

		const result: ApiResponse<T> = await response.json();
		if (!result.success) {
			throw new Error(result.error || 'Unknown error');
		}

		return result.data!;
	},

	async insertRecord<T = TableRecord>(tableName: string, data: Partial<T>): Promise<T> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/insert`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});

		if (!response.ok) {
			throw new Error(`Failed to insert ${tableName}: ${response.statusText}`);
		}

		const result: ApiResponse<T> = await response.json();
		if (!result.success) {
			throw new Error(result.error || 'Unknown error');
		}

		return result.data!;
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

		if (!response.ok) {
			throw new Error(`Failed to modify ${tableName} ${id}: ${response.statusText}`);
		}

		const result: ApiResponse<T> = await response.json();
		if (!result.success) {
			throw new Error(result.error || 'Unknown error');
		}

		return result.data!;
	},

	async deleteRecord(tableName: string, id: string): Promise<void> {
		const response = await fetch(`${API_BASE}/tables/${tableName}/delete/${id}`, {
			method: 'DELETE'
		});

		if (!response.ok) {
			throw new Error(`Failed to delete ${tableName} ${id}: ${response.statusText}`);
		}

		const result: ApiResponse = await response.json();
		if (!result.success) {
			throw new Error(result.error || 'Unknown error');
		}
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

		if (!response.ok) {
			throw new Error(`Failed to run codeunit ${codeunitId}: ${response.statusText}`);
		}

		const result: ApiResponse = await response.json();
		if (!result.success) {
			throw new Error(result.error || 'Unknown error');
		}

		return result.data;
	},

	// Authentication
	async login(userID: string, password: string, company?: string): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/login`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ user_id: userID, password, company })
		});

		if (!response.ok) {
			const result: ApiResponse = await response.json();
			throw { status: response.status, message: result.error || 'Login failed' };
		}

		return await response.json();
	},

	async logout(): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/logout`, {
			method: 'POST'
		});

		if (!response.ok) {
			throw new Error('Logout failed');
		}

		return await response.json();
	},

	async getCurrentUser(): Promise<ApiResponse> {
		const response = await fetch(`${API_BASE}/auth/user`);

		if (!response.ok) {
			throw { status: response.status, message: 'Not authenticated' };
		}

		return await response.json();
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

		if (!response.ok) {
			const result: ApiResponse = await response.json();
			throw new Error(result.error || 'Failed to create user');
		}

		return await response.json();
	},

	async listCompanies(): Promise<ApiResponse<string[]>> {
		const response = await fetch(`${API_BASE}/auth/companies`);

		if (!response.ok) {
			throw new Error('Failed to list companies');
		}

		return await response.json();
	}
};

// Specific table APIs (typed)
export const customerApi = {
	list: (options?: ListOptions) => api.listRecords('Customer', options),
	get: (no: string) => api.getRecord('Customer', no),
	insert: (data: any) => api.insertRecord('Customer', data),
	modify: (no: string, data: any) => api.modifyRecord('Customer', no, data),
	delete: (no: string) => api.deleteRecord('Customer', no)
};

export const paymentTermsApi = {
	list: (options?: ListOptions) => api.listRecords('Payment_terms', options),
	get: (code: string) => api.getRecord('Payment_terms', code),
	insert: (data: any) => api.insertRecord('Payment_terms', data),
	modify: (code: string, data: any) => api.modifyRecord('Payment_terms', code, data),
	delete: (code: string) => api.deleteRecord('Payment_terms', code)
};
