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
