// The URL of an action's run_page with its filters from the current record: run_page_filter_field
// = record[run_page_filter_value] (one field), or run_page_filters — several fields, each from
// the record (value), a constant (const) or the range of two of its fields (from..to — a
// register's entries), classic RunFormLink. null when a filter needs a value the record does
// not have (no row selected).

export interface RunPageAction {
	run_page?: number;
	run_page_filter_field?: string;
	run_page_filter_value?: string;
	run_page_filters?: { field: string; value?: string; const?: string; from?: string; to?: string }[];
}

export function runPagePath(action: RunPageAction, record: Record<string, any> | undefined | null): string | null {
	const filters: string[] = [];
	if (action.run_page_filter_field) {
		const value = action.run_page_filter_value ? record?.[action.run_page_filter_value] : undefined;
		if (value === undefined || value === null || value === '') return null;
		filters.push(`${action.run_page_filter_field}=${value}`);
	}
	for (const f of action.run_page_filters ?? []) {
		if (f.from !== undefined && f.to !== undefined) {
			const from = record?.[f.from];
			const to = record?.[f.to];
			if (from === undefined || from === null || from === '' || to === undefined || to === null || to === '') return null;
			filters.push(`${f.field}=${from}..${to}`);
			continue;
		}
		const value = f.value !== undefined ? record?.[f.value] : f.const;
		if (value === undefined || value === null || value === '') return null;
		filters.push(`${f.field}=${value}`);
	}
	const query = filters.map((f) => `filter=${encodeURIComponent(f)}`).join('&');
	return `/pages/${action.run_page}` + (query ? `?${query}` : '');
}
