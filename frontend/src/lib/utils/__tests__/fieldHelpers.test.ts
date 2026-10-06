import { describe, it, expect } from 'vitest';
import { isAdvancedLookup, formatLookupValue } from '../fieldHelpers';

describe('isAdvancedLookup', () => {
	const columns = [{ source: 'code' }];
	it('is a table-style dropdown with columns and rows, or columns and a lazy URL', () => {
		expect(isAdvancedLookup({ columns, rows: [{ _key: 'A' }] })).toBe(true);
		expect(isAdvancedLookup({ columns, lazy_url: '/api/tables/X/lookup/f' })).toBe(true);
		expect(isAdvancedLookup({ columns, rows: null, lazy_url: '/api/tables/X/lookup/f' })).toBe(true);
	});
	it('is not without columns, or without rows and lazy URL', () => {
		expect(isAdvancedLookup(undefined)).toBe(false);
		expect(isAdvancedLookup({ columns: [], rows: [{ _key: 'A' }] })).toBe(false);
		expect(isAdvancedLookup({ columns, rows: [] })).toBe(false);
	});
});

describe('formatLookupValue', () => {
	it('shows the key when on-demand rows are not loaded', () => {
		expect(formatLookupValue('C00010', { columns: [{ source: 'no' }] })).toBe('C00010');
	});
});
